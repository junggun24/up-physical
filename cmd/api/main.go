// 인제스션 & 결과 조회 API 서버 (Contract-first, api/openapi.yaml).
//
// 역할: 골격 스트림 업로드를 받아 (1) 계약 스키마·불변식 검증(경계 방어),
// (2) 오브젝트 스토리지 저장, (3) 세션/subjects/잡 기록(Postgres), (4) 즉시 job_id 반환(202).
// 처리는 워커(cmd/worker)가 비동기로 수행한다.
//
// 엔드포인트:
//   GET  /healthz
//   GET  /v1/references?sport=
//   POST /v1/sessions                     (Idempotency-Key 헤더 필수)
//   GET  /v1/sessions/{session_id}
//   GET  /v1/jobs/{job_id}
//   GET  /v1/jobs/{job_id}/results
//
// Python 참조구현: ../../../services/ingestion.py
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/upphysical/backend/internal/auth"
	"github.com/upphysical/backend/internal/contract"
	"github.com/upphysical/backend/internal/objstore"
	"github.com/upphysical/backend/internal/store"
)

const maxBodyBytes = 16 << 20 // 16MB (골격 스트림은 보통 수십 KB~수 MB)

type server struct {
	st           *store.Store
	obj          *objstore.Store
	hub          *hub
	jwtSecret    string
	allowDevAuth bool
}

const tokenTTL = 720 * time.Hour // 30일

func main() {
	ctx := context.Background()

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

	h := newHub()
	go h.listen(ctx, st.Pool)

	s := &server{
		st:           st,
		obj:          obj,
		hub:          h,
		jwtSecret:    mustEnv("JWT_SECRET"),
		allowDevAuth: os.Getenv("ALLOW_DEV_AUTH") == "true",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("POST /v1/auth/signup", s.handleSignup)
	mux.HandleFunc("POST /v1/auth/login", s.handleLogin)
	mux.HandleFunc("GET /v1/references", s.handleListReferences)
	mux.HandleFunc("GET /v1/sessions", s.handleListSessions)
	mux.HandleFunc("POST /v1/sessions", s.handleCreateSession)
	mux.HandleFunc("GET /v1/sessions/{session_id}", s.handleGetSession)
	mux.HandleFunc("GET /v1/jobs/{job_id}", s.handleGetJob)
	mux.HandleFunc("GET /v1/jobs/{job_id}/events", s.handleJobEvents)
	mux.HandleFunc("GET /v1/jobs/{job_id}/results", s.handleGetResults)

	addr := ":" + envOr("PORT", "8080")
	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Printf("upphysical ingestion api listening on %s", addr)
	log.Fatal(srv.ListenAndServe())
}

// ─────────────────────────── 핸들러 ───────────────────────────

func (s *server) handleListReferences(w http.ResponseWriter, r *http.Request) {
	refs, err := s.st.ListReferences(r.Context(), r.URL.Query().Get("sport"))
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	if refs == nil {
		refs = []store.ReferenceCatalog{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"references": refs})
}

type ingestRequest struct {
	Stream   json.RawMessage `json:"stream"`
	Analysis struct {
		Sport            string `json:"sport"`
		Action           string `json:"action"`
		ReferenceVersion *int   `json:"reference_version"`
		// Handedness — "right" | "left" (선택). 손잡이가 레퍼런스와 다르면 워커가
		// 좌우 반전으로 정규화한다. 없으면 정규화를 건너뛴다.
		Handedness string `json:"handedness"`
	} `json:"analysis"`
}

func (s *server) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idemKey := r.Header.Get("Idempotency-Key")
	if idemKey == "" {
		writeProblem(w, http.StatusBadRequest, "missing_idempotency_key", "Idempotency-Key 헤더 필수")
		return
	}
	userID, ok := s.resolveUserID(ctx, r)
	if !ok {
		writeProblem(w, http.StatusUnauthorized, "unauthorized", "인증 필요(로그인)")
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, maxBodyBytes+1))
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "read_error", err.Error())
		return
	}
	if len(body) > maxBodyBytes {
		writeProblem(w, http.StatusRequestEntityTooLarge, "payload_too_large", "본문이 너무 큽니다")
		return
	}
	var req ingestRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeProblem(w, http.StatusBadRequest, "bad_json", err.Error())
		return
	}
	if req.Analysis.Sport == "" || req.Analysis.Action == "" {
		writeProblem(w, http.StatusBadRequest, "bad_request", "analysis.sport, analysis.action 필수")
		return
	}
	if h := req.Analysis.Handedness; h != "" && h != "right" && h != "left" {
		writeProblem(w, http.StatusBadRequest, "bad_request", "analysis.handedness 는 right|left 만 허용")
		return
	}

	// 1) 경계 검증(계약 스키마 구조 + 불변식)
	st, err := contract.Parse(req.Stream)
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_stream", err.Error())
		return
	}
	if verrs := contract.Validate(st); len(verrs) > 0 {
		writeProblemWithErrors(w, http.StatusBadRequest, "invalid_stream", "골격 스트림 검증 실패", verrs)
		return
	}

	// 2) 멱등성: 같은 키면 기존 잡 반환
	if existing, err := s.st.FindJobByIdempotency(ctx, idemKey); err != nil {
		writeProblem(w, http.StatusInternalServerError, "internal", err.Error())
		return
	} else if existing != nil {
		writeJSON(w, http.StatusAccepted, map[string]any{
			"session_id": existing.SessionID, "job_id": existing.ID, "status": existing.Status,
		})
		return
	}

	// 3) 레퍼런스 해석: 활성 레퍼런스
	ref, err := s.st.ResolveActiveReference(ctx, req.Analysis.Sport, req.Analysis.Action)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	if ref == nil {
		writeProblem(w, http.StatusUnprocessableEntity, "reference_not_found",
			fmt.Sprintf("활성 레퍼런스 없음: %s/%s", req.Analysis.Sport, req.Analysis.Action))
		return
	}

	// 4) 원본 스트림 저장(검증된 바이트 그대로 보존) — 성공 후에만 메타 커밋
	key := fmt.Sprintf("sessions/%s/stream.json", st.SessionID)
	meta, err := s.obj.PutBytes(ctx, store.BucketStreams, key, req.Stream)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "storage_error", err.Error())
		return
	}

	// 6) 세션/subjects/잡 기록(트랜잭션)
	jobID, err := s.st.CreateSessionWithJob(ctx, st, userID, ref.ID, idemKey,
		store.BucketStreams, key, meta.Bytes, req.Analysis.Handedness)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]any{
		"session_id": st.SessionID, "job_id": jobID, "status": "queued",
	})
}

func (s *server) handleListSessions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := s.resolveUserID(ctx, r)
	if !ok {
		writeProblem(w, http.StatusUnauthorized, "unauthorized", "인증 필요(로그인)")
		return
	}
	sessions, err := s.st.ListUserSessions(ctx, userID)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	if sessions == nil {
		sessions = []store.SessionSummary{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"sessions": sessions})
}

func (s *server) handleGetSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("session_id")
	var status string
	err := s.st.Pool.QueryRow(r.Context(),
		`SELECT status FROM sessions WHERE id=$1::uuid`, id).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		writeProblem(w, http.StatusNotFound, "not_found", "세션 없음")
		return
	}
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"session_id": id, "status": status})
}

func (s *server) handleGetJob(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("job_id")
	status, err := s.st.JobStatus(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		writeProblem(w, http.StatusNotFound, "not_found", "잡 없음")
		return
	}
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"job_id": id, "status": status})
}

func (s *server) handleGetResults(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("job_id")
	results, err := s.st.GetResults(r.Context(), id)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	if results == nil {
		results = []map[string]any{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"job_id": id, "results": results})
}

// ─────────────────────────── 헬퍼 ───────────────────────────

// ─────────────────────────── 인증 ───────────────────────────

type authReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *server) handleSignup(w http.ResponseWriter, r *http.Request) {
	var req authReq
	if !decodeAuth(w, r, &req) {
		return
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	id, err := s.st.CreateLocalUser(r.Context(), req.Email, hash)
	if err != nil {
		if store.IsUniqueViolation(err) {
			writeProblem(w, http.StatusConflict, "email_taken", "이미 가입된 이메일입니다")
			return
		}
		writeProblem(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	issueToken(w, s.jwtSecret, id, req.Email)
}

func (s *server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req authReq
	if !decodeAuth(w, r, &req) {
		return
	}
	id, hash, err := s.st.GetLocalUserByEmail(r.Context(), req.Email)
	if err != nil || hash == "" || !auth.CheckPassword(hash, req.Password) {
		writeProblem(w, http.StatusUnauthorized, "invalid_credentials", "이메일 또는 비밀번호가 올바르지 않습니다")
		return
	}
	issueToken(w, s.jwtSecret, id, req.Email)
}

func decodeAuth(w http.ResponseWriter, r *http.Request, req *authReq) bool {
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(req); err != nil {
		writeProblem(w, http.StatusBadRequest, "bad_json", err.Error())
		return false
	}
	req.Email = strings.TrimSpace(req.Email)
	if req.Email == "" || !strings.Contains(req.Email, "@") {
		writeProblem(w, http.StatusBadRequest, "bad_email", "유효한 이메일을 입력하세요")
		return false
	}
	if len(req.Password) < 6 {
		writeProblem(w, http.StatusBadRequest, "weak_password", "비밀번호는 6자 이상이어야 합니다")
		return false
	}
	return true
}

func issueToken(w http.ResponseWriter, secret, userID, email string) {
	tok, err := auth.Sign(userID, email, secret, tokenTTL)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"token": tok, "user_id": userID})
}

// resolveUserID — 요청에서 인증 주체(users.id)를 해석.
// (a) Bearer JWT 검증 → sub(=users.id),  (b) 개발 모드면 X-User-Id → 로컬 dev 유저로 매핑.
func (s *server) resolveUserID(ctx context.Context, r *http.Request) (string, bool) {
	authz := r.Header.Get("Authorization")
	if strings.HasPrefix(authz, "Bearer ") {
		claims, err := auth.Verify(strings.TrimPrefix(authz, "Bearer "), s.jwtSecret)
		if err != nil {
			return "", false
		}
		return claims.Sub, true
	}
	if s.allowDevAuth {
		if v := r.Header.Get("X-User-Id"); v != "" {
			id, err := s.st.EnsureUser(ctx, "dev:"+v, nil)
			if err != nil {
				return "", false
			}
			return id, true
		}
	}
	return "", false
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeProblem(w http.ResponseWriter, code int, typ, detail string) {
	writeProblemWithErrors(w, code, typ, detail, nil)
}

func writeProblemWithErrors(w http.ResponseWriter, code int, typ, detail string, verrs []contract.ValErr) {
	w.Header().Set("Content-Type", "application/problem+json; charset=utf-8")
	w.WriteHeader(code)
	body := map[string]any{"type": typ, "status": code, "detail": detail}
	if len(verrs) > 0 {
		body["errors"] = verrs
	}
	_ = json.NewEncoder(w).Encode(body)
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

// parseEndpoint — "http(s)://host:port" → (host:port, ssl). minio-go 는 스킴 없는 호스트를 받는다.
func parseEndpoint(raw string) (host string, ssl bool) {
	ssl = strings.HasPrefix(raw, "https://")
	host = strings.TrimPrefix(strings.TrimPrefix(raw, "https://"), "http://")
	host = strings.TrimSuffix(host, "/")
	return host, ssl
}
