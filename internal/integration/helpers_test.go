// Package integration — 기동된 스택(Postgres·MinIO·API·워커·엔진) 대상 통합 검증.
//
// 왜 필요한가: 채점 경로의 결함(레퍼런스 최고점 채택)과 손잡이 정규화는 유닛으로 잡히지
// 않는다 — DB·스토리지·큐·엔진이 함께 있어야 드러나는 동작이기 때문이다. 2026-07-30 에
// 이 둘을 수동 절차로 확인했는데, 수동 절차는 재발을 감지하지 못한다.
//
// 실행: .harness/runners/check-integration.sh  (UPX_INTEGRATION=1 이 없으면 전부 skip)
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/upphysical/backend/internal/contract"
	"github.com/upphysical/backend/internal/objstore"
	"github.com/upphysical/backend/internal/store"
)

const (
	pollTimeout  = 60 * time.Second
	pollInterval = 300 * time.Millisecond
)

type env struct {
	st      *store.Store
	obj     *objstore.Store
	apiBase string
	repoRoot string
}

// setup — 통합 실행 조건을 확인하고 의존성을 연결한다. 조건 미충족이면 skip(실패 아님).
func setup(t *testing.T) *env {
	t.Helper()
	if os.Getenv("UPX_INTEGRATION") != "1" {
		t.Skip("통합 테스트 생략 — .harness/runners/check-integration.sh 로 실행하세요")
	}
	ctx := context.Background()

	dsn := mustEnv(t, "DATABASE_URL")
	st, err := store.New(ctx, dsn)
	if err != nil {
		t.Fatalf("DB 연결 실패(스택이 떠 있나요?): %v", err)
	}
	t.Cleanup(st.Close)

	host, ssl := parseEndpoint(mustEnv(t, "S3_ENDPOINT"))
	obj, err := objstore.New(objstore.Config{
		Endpoint:  host,
		AccessKey: mustEnv(t, "S3_ACCESS_KEY"),
		SecretKey: mustEnv(t, "S3_SECRET_KEY"),
		UseSSL:    ssl,
	})
	if err != nil {
		t.Fatalf("objstore 연결 실패: %v", err)
	}

	base := envOr("UPX_API_BASE", "http://localhost:8080")
	resp, err := http.Get(base + "/healthz")
	if err != nil {
		t.Fatalf("API 응답 없음 %s (API·워커가 떠 있나요?): %v", base, err)
	}
	_ = resp.Body.Close()

	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	return &env{st: st, obj: obj, apiBase: base, repoRoot: root}
}

func mustEnv(t *testing.T, k string) string {
	t.Helper()
	v := os.Getenv(k)
	if v == "" {
		t.Fatalf("환경변수 %s 필요 — set -a; source deploy/.env; set +a", k)
	}
	return v
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func parseEndpoint(raw string) (string, bool) {
	ssl := strings.HasPrefix(raw, "https://")
	h := strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(raw, "https://"), "http://"), "/")
	return h, ssl
}

// fixture — 리포의 fixture 를 읽어 파싱한다.
func (e *env) fixture(t *testing.T, name string) *contract.Stream {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(e.repoRoot, ".harness", "fixtures", name))
	if err != nil {
		t.Fatalf("fixture 읽기 실패: %v", err)
	}
	st, err := contract.Parse(raw)
	if err != nil {
		t.Fatalf("fixture 파싱 실패: %v", err)
	}
	return st
}

// seedReference — 테스트 전용 동작(action)에 레퍼런스를 등록하고 활성화한다.
// 같은 action 에 두 번 부르면 나중 것이 활성이 되고 이전 것은 남는다(비활성).
func (e *env) seedReference(t *testing.T, action string, version int, st *contract.Stream, handedness string) string {
	t.Helper()
	ctx := context.Background()

	raw, err := json.Marshal(st)
	if err != nil {
		t.Fatal(err)
	}
	key := fmt.Sprintf("test/%s/v%d/reference.json", action, version)
	if _, err := e.obj.PutBytes(ctx, store.BucketReferences, key, raw); err != nil {
		t.Fatalf("레퍼런스 업로드 실패: %v", err)
	}
	id, err := e.st.RegisterReference(ctx, "test", action, version, st, store.BucketReferences, key,
		store.ReferenceMeta{
			SourceKind:  "synthetic",
			RightsBasis: "통합 테스트 합성 데이터 — 저작물 아님",
			Handedness:  handedness,
		})
	if err != nil {
		t.Fatalf("레퍼런스 등록 실패: %v", err)
	}
	return id
}

// upload — 스트림을 업로드하고 job_id 를 돌려준다(폴링하지 않는다).
func (e *env) upload(t *testing.T, action string, st *contract.Stream,
	handedness, idemKey, sessionID string) string {
	t.Helper()

	fresh := *st
	fresh.SessionID = sessionID
	streamJSON, err := json.Marshal(&fresh)
	if err != nil {
		t.Fatal(err)
	}

	analysis := map[string]any{"sport": "test", "action": action}
	if handedness != "" {
		analysis["handedness"] = handedness
	}
	body, err := json.Marshal(map[string]any{
		"stream":   json.RawMessage(streamJSON),
		"analysis": analysis,
	})
	if err != nil {
		t.Fatal(err)
	}

	req, _ := http.NewRequest(http.MethodPost, e.apiBase+"/v1/sessions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", idemKey)
	req.Header.Set("X-User-Id", "integration-test")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("업로드 실패: %v", err)
	}
	defer res.Body.Close()
	payload, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusAccepted {
		t.Fatalf("업로드 응답 %d: %s", res.StatusCode, payload)
	}
	var created struct {
		JobID string `json:"job_id"`
	}
	if err := json.Unmarshal(payload, &created); err != nil {
		t.Fatalf("응답 파싱 실패: %s", payload)
	}
	return created.JobID
}

// analyze — 업로드 → 완료까지 폴링 → 첫 피사체의 종합 점수.
func (e *env) analyze(t *testing.T, action string, st *contract.Stream, handedness string) float64 {
	t.Helper()
	// 세션 id 는 매번 새로 (같은 id 재사용은 별건의 알려진 버그다)
	jobID := e.upload(t, action, st, handedness, uuid.NewString(), uuid.NewString())
	if status := e.pollJob(t, jobID); status != "succeeded" {
		t.Fatalf("잡이 성공하지 않음: %s (job %s)", status, jobID)
	}
	return e.firstScore(t, jobID)
}

// analyzeTwiceSameKey — 같은 멱등키·같은 본문으로 두 번 업로드한다(클라이언트 재시도 재현).
func (e *env) analyzeTwiceSameKey(t *testing.T, action string, st *contract.Stream) (string, string) {
	t.Helper()
	key, session := uuid.NewString(), uuid.NewString()
	first := e.upload(t, action, st, "", key, session)
	if status := e.pollJob(t, first); status != "succeeded" {
		t.Fatalf("첫 잡이 성공하지 않음: %s", status)
	}
	second := e.upload(t, action, st, "", key, session)
	return first, second
}

func (e *env) pollJob(t *testing.T, jobID string) string {
	t.Helper()
	deadline := time.Now().Add(pollTimeout)
	for time.Now().Before(deadline) {
		res, err := http.Get(e.apiBase + "/v1/jobs/" + jobID)
		if err != nil {
			t.Fatalf("잡 조회 실패: %v", err)
		}
		var got struct {
			Status string `json:"status"`
		}
		b, _ := io.ReadAll(res.Body)
		_ = res.Body.Close()
		_ = json.Unmarshal(b, &got)
		if got.Status == "succeeded" || got.Status == "failed" {
			return got.Status
		}
		time.Sleep(pollInterval)
	}
	t.Fatalf("잡이 %v 안에 끝나지 않음 (워커가 떠 있나요?)", pollTimeout)
	return ""
}

func (e *env) firstScore(t *testing.T, jobID string) float64 {
	t.Helper()
	res, err := http.Get(e.apiBase + "/v1/jobs/" + jobID + "/results")
	if err != nil {
		t.Fatalf("결과 조회 실패: %v", err)
	}
	defer res.Body.Close()
	var got struct {
		Results []struct {
			OverallScore float64 `json:"overall_score"`
		} `json:"results"`
	}
	b, _ := io.ReadAll(res.Body)
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("결과 파싱 실패: %s", b)
	}
	if len(got.Results) == 0 {
		t.Fatalf("결과가 비어 있음 (job %s)", jobID)
	}
	return got.Results[0].OverallScore
}

// uniqueAction — 실행마다 격리된 동작 이름 (실제 tennis/forehand 레퍼런스를 건드리지 않는다).
func uniqueAction(prefix string) string {
	return fmt.Sprintf("%s-%s", prefix, strings.ReplaceAll(uuid.NewString()[:8], "-", ""))
}

// cleanupAction — 테스트가 만든 레퍼런스·잡·세션을 지운다.
func (e *env) cleanupAction(t *testing.T, action string) {
	t.Helper()
	ctx := context.Background()
	// 잡 → 세션 순으로 지운다 (results·subjects 는 CASCADE).
	_, _ = e.st.Pool.Exec(ctx, `
		DELETE FROM sessions WHERE id IN (
			SELECT s.id FROM sessions s
			JOIN analysis_jobs j ON j.session_id = s.id
			JOIN reference_streams rs ON rs.id = j.reference_id
			JOIN reference_actions ra ON ra.id = rs.action_id
			WHERE ra.sport='test' AND ra.action=$1)`, action)
	_, _ = e.st.Pool.Exec(ctx, `
		DELETE FROM reference_streams WHERE action_id IN (
			SELECT id FROM reference_actions WHERE sport='test' AND action=$1)`, action)
	_, _ = e.st.Pool.Exec(ctx, `DELETE FROM reference_actions WHERE sport='test' AND action=$1`, action)
}
