// 분석 워커.
//
// 역할: 큐(Postgres)에서 잡을 선점 → 스트림+레퍼런스 로드(오브젝트 스토리지)
// → 분석 엔진 호출(정규화·DTW·채점·피드백) → 결과 저장. at-least-once + 멱등 처리.
//
// Python 참조구현: ../../../services/worker.py, 알고리즘: ../../../engine/upphysical_engine/
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/upphysical/backend/internal/analysis"
	"github.com/upphysical/backend/internal/contract"
	"github.com/upphysical/backend/internal/normalize"
	"github.com/upphysical/backend/internal/objstore"
	"github.com/upphysical/backend/internal/queue"
	"github.com/upphysical/backend/internal/store"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	st, err := store.New(ctx, mustEnv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("store 초기화 실패: %v", err)
	}
	defer st.Close()

	s3host, s3ssl := parseEndpoint(mustEnv("S3_ENDPOINT"))
	obj, err := objstore.New(objstore.Config{
		Endpoint:  s3host,
		AccessKey: mustEnv("S3_ACCESS_KEY"),
		SecretKey: mustEnv("S3_SECRET_KEY"),
		UseSSL:    s3ssl,
	})
	if err != nil {
		log.Fatalf("objstore 초기화 실패: %v", err)
	}

	engine := analysis.NewEngine(envOr("ENGINE_PYTHON", "python3"), mustEnv("ENGINE_SCRIPT"))
	defer engine.Stop()

	log.Printf("upphysical analysis worker started")
	idle := 400 * time.Millisecond // 잡 없을 때 짧게 폴링

	for {
		select {
		case <-ctx.Done():
			log.Printf("graceful shutdown")
			return
		default:
		}

		claimed, err := queue.ClaimNext(ctx, st.Pool)
		if err != nil {
			log.Printf("claim 오류: %v", err)
			sleep(ctx, idle)
			continue
		}
		if claimed == nil {
			sleep(ctx, idle) // 큐 비어 있음
			continue
		}
		tStart := time.Now()
		if err := process(ctx, st, obj, engine, claimed); err != nil {
			log.Printf("[job %s] 실패 (%dms): %v", short(claimed.JobID), time.Since(tStart).Milliseconds(), err)
		} else {
			log.Printf("[job %s] 완료 (%dms)", short(claimed.JobID), time.Since(tStart).Milliseconds())
		}
	}
}

func process(ctx context.Context, st *store.Store, obj *objstore.Store,
	engine *analysis.Engine, c *queue.Claimed) error {

	fail := func(e error) error {
		_ = st.MarkJobFailed(ctx, c.JobID, c.SessionID, e.Error())
		return e
	}

	// 1) 사용자 스트림 + 레퍼런스 로드
	bucket, key, userHand, err := st.GetSessionStreamLoc(ctx, c.SessionID)
	if err != nil {
		return fail(err)
	}
	userJSON, err := obj.GetBytes(ctx, bucket, key)
	if err != nil {
		return fail(err)
	}

	// 잡 생성 시 고정된 **대표 레퍼런스 1개**로만 채점한다.
	//
	// 과거엔 동작의 모든 레퍼런스를 돌려 최고점을 택했는데, 그러면 레퍼런스가 늘수록
	// 모든 사용자의 점수가 단조 증가하고(max), 피드백이 "당신의 결함"이 아니라
	// "당신과 가장 닮은 코치"가 되어 '고칠 단 하나'의 근거가 사라진다.
	// 엔진 호출도 뮤텍스 직렬화라 레퍼런스 N개면 지연이 N배(p50 예산 초과).
	// 다중 레퍼런스 비교는 채점이 아니라 별도 기능(오버레이·매칭)으로 다룬다.
	if c.ReferenceID == "" {
		return fail(fmt.Errorf("잡에 레퍼런스가 없음"))
	}
	ref, err := st.GetReferenceByID(ctx, c.ReferenceID)
	if err != nil {
		return fail(fmt.Errorf("레퍼런스 로드 실패: %w", err))
	}
	refJSON, err := obj.GetBytes(ctx, ref.Bucket, ref.Key)
	if err != nil {
		return fail(err)
	}

	// 1-1) 손잡이 정규화 — 사용자와 레퍼런스의 손잡이가 다르면 좌우를 반전한다.
	// 반전 없이 비교하면 점수가 "자세 차이"가 아니라 "손잡이 차이"를 반영한다.
	// 저장된 원본은 건드리지 않고, 엔진에 넘길 바이트만 변환한다.
	if userHand != "" && ref.Handedness != "" && userHand != ref.Handedness {
		mirrored, mErr := mirrorStreamJSON(userJSON)
		if mErr != nil {
			return fail(fmt.Errorf("손잡이 정규화 실패: %w", mErr))
		}
		userJSON = mirrored
		log.Printf("[job %s] 손잡이 정규화 적용 (user=%s, ref=%s)", short(c.JobID), userHand, ref.Handedness)
	}

	// 2) 분석 엔진 (상주 프로세스, ref 는 reference_id 로 캐시)
	tEng := time.Now()
	res, err := engine.Analyze(ctx, userJSON, refJSON, ref.ID)
	if err != nil {
		return fail(err)
	}
	log.Printf("[job %s] 엔진 %dms (%d subjects, ref %s)", short(c.JobID), time.Since(tEng).Milliseconds(), len(res.Results), short(ref.ID))

	// 3) subject_key → subjects.id 매핑 후 결과 저장(멱등)
	smap, err := st.SubjectsMap(ctx, c.SessionID)
	if err != nil {
		return fail(err)
	}
	for _, r := range res.Results {
		sid, ok := smap[r.SubjectKey]
		if !ok {
			continue
		}
		breakdown := jsonOrDefault(r.ScoreBreakdown, "{}")
		feedback := jsonOrDefault(r.Feedback, "[]")
		if err := st.UpsertResult(ctx, c.JobID, sid, r.OverallScore, r.DTWDistance, breakdown, feedback, []byte(res.Comparison)); err != nil {
			return fail(err)
		}
	}

	// 4) 성공 마킹
	return st.MarkJobSucceeded(ctx, c.JobID, c.SessionID)
}

// mirrorStreamJSON — 스트림 바이트를 좌우 반전한 바이트로 바꾼다(원본 바이트는 보존).
func mirrorStreamJSON(raw []byte) ([]byte, error) {
	st, err := contract.Parse(raw)
	if err != nil {
		return nil, err
	}
	return json.Marshal(normalize.MirrorStream(st))
}

func jsonOrDefault(raw json.RawMessage, def string) []byte {
	if len(raw) == 0 {
		return []byte(def)
	}
	return raw
}

func sleep(ctx context.Context, d time.Duration) {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
	case <-t.C:
	}
}

func short(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

func mustEnv(k string) string {
	v := os.Getenv(k)
	if v == "" {
		log.Fatalf("환경변수 %s 필요", k)
	}
	return v
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func parseEndpoint(raw string) (host string, ssl bool) {
	ssl = strings.HasPrefix(raw, "https://")
	host = strings.TrimPrefix(strings.TrimPrefix(raw, "https://"), "http://")
	host = strings.TrimSuffix(host, "/")
	return host, ssl
}
