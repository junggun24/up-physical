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
	bucket, key, err := st.GetSessionStreamLoc(ctx, c.SessionID)
	if err != nil {
		return fail(err)
	}
	userJSON, err := obj.GetBytes(ctx, bucket, key)
	if err != nil {
		return fail(err)
	}

	// 동작의 모든 레퍼런스(코치 스윙) 로드 — 여럿이면 best 점수를 택해 단일 레퍼런스 분산 완화.
	refs, err := st.ReferencesForAction(ctx, c.ReferenceID)
	if err != nil || len(refs) == 0 {
		return fail(fmt.Errorf("레퍼런스 로드 실패: %v", err))
	}

	// 2) 분석 엔진 (상주 프로세스, ref 는 reference_id 로 캐시)
	tEng := time.Now()
	var res *analysis.Result
	for _, rf := range refs {
		refJSON, e := obj.GetBytes(ctx, rf.Bucket, rf.Key)
		if e != nil {
			continue
		}
		r, e := engine.Analyze(ctx, userJSON, refJSON, rf.ID)
		if e != nil {
			continue
		}
		if res == nil || primaryScore(r) > primaryScore(res) {
			res = r
		}
	}
	if res == nil {
		return fail(fmt.Errorf("분석 결과 없음(레퍼런스 %d개)", len(refs)))
	}
	log.Printf("[job %s] 엔진 %dms (%d subjects, ref %d개)", short(c.JobID), time.Since(tEng).Milliseconds(), len(res.Results), len(refs))

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

// primaryScore — 주 피사체(첫 결과)의 종합 점수. 다중 레퍼런스 중 best 선택 기준.
func primaryScore(r *analysis.Result) float64 {
	if r == nil || len(r.Results) == 0 {
		return -1
	}
	return r.Results[0].OverallScore
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
