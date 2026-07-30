// Package store — PostgreSQL 데이터 계층 (pgx).
//
// 강한 제약(FK/UNIQUE/CHECK)은 DB가 강제(db/migrations/0001_init.up.sql).
// 참조구현(services/{db,ingestion,worker,references}.py)의 SQL을 pgx 로 이식.
// 잡 큐는 별도 인프라 없이 Postgres(SELECT … FOR UPDATE SKIP LOCKED)로 처리 → internal/queue.
package store

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/upphysical/backend/internal/contract"
)

const (
	BucketStreams    = "skeleton-streams"
	BucketReferences = "references"
)

type Store struct{ Pool *pgxpool.Pool }

func New(ctx context.Context, dsn string) (*Store, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("pgxpool 생성 실패: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("DB 핑 실패: %w", err)
	}
	return &Store{Pool: pool}, nil
}

func (s *Store) Close() { s.Pool.Close() }

// ─────────────────────────── 모델 ───────────────────────────

type Job struct {
	ID          string
	SessionID   string
	ReferenceID string
	Status      string
}

type Reference struct {
	ID         string
	Bucket     string
	Key        string
	Handedness string // "right" | "left" | "" (미지정)
}

type ReferenceCatalog struct {
	Sport   string `json:"sport"`
	Action  string `json:"action"`
	Version int    `json:"version"`
}

type Result struct {
	SubjectKey     string  `json:"subject_key"`
	OverallScore   float64 `json:"overall_score"`
	DTWDistance    float64 `json:"dtw_distance"`
	ScoreBreakdown any     `json:"score_breakdown"`
	Feedback       any     `json:"feedback"`
}

// ─────────────────────────── 쓰기 ───────────────────────────

// EnsureUser — external_id(IdP sub) 기준 upsert. 반환: user id.
func (s *Store) EnsureUser(ctx context.Context, externalID string, email *string) (string, error) {
	var id string
	err := s.Pool.QueryRow(ctx,
		`SELECT id::text FROM users WHERE external_id=$1`, externalID).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}
	newID := uuid.NewString()
	_, err = s.Pool.Exec(ctx,
		`INSERT INTO users(id, external_id, email) VALUES($1::uuid,$2,$3)`, newID, externalID, email)
	if err != nil {
		return "", err
	}
	return newID, nil
}

// CreateLocalUser — 자체 인증 회원가입. external_id='local:'+lower(email). 반환: user id.
// 이메일 중복이면 UNIQUE 위반 에러를 그대로 돌려준다(핸들러가 409로 매핑).
func (s *Store) CreateLocalUser(ctx context.Context, email, passwordHash string) (string, error) {
	id := uuid.NewString()
	_, err := s.Pool.Exec(ctx,
		`INSERT INTO users(id, external_id, email, password_hash) VALUES($1::uuid,$2,$3,$4)`,
		id, "local:"+strings.ToLower(email), email, passwordHash)
	if err != nil {
		return "", err
	}
	return id, nil
}

// GetLocalUserByEmail — 로그인용. 반환: (user id, password_hash). 없으면 pgx.ErrNoRows.
func (s *Store) GetLocalUserByEmail(ctx context.Context, email string) (string, string, error) {
	var id, hash string
	err := s.Pool.QueryRow(ctx,
		`SELECT id::text, COALESCE(password_hash,'') FROM users WHERE lower(email)=lower($1)`, email).
		Scan(&id, &hash)
	return id, hash, err
}

// IsUniqueViolation — Postgres UNIQUE 제약 위반(중복 이메일 등) 여부.
func IsUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "SQLSTATE 23505")
}

// FindJobByIdempotency — 같은 키의 잡이 있으면 반환(멱등). 없으면 (nil,nil).
func (s *Store) FindJobByIdempotency(ctx context.Context, key string) (*Job, error) {
	var j Job
	err := s.Pool.QueryRow(ctx,
		`SELECT id::text, session_id::text, status FROM analysis_jobs WHERE idempotency_key=$1`, key).
		Scan(&j.ID, &j.SessionID, &j.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &j, nil
}

// ResolveActiveReference — (sport, action)의 활성 레퍼런스. 없으면 (nil,nil).
func (s *Store) ResolveActiveReference(ctx context.Context, sport, action string) (*Reference, error) {
	var r Reference
	err := s.Pool.QueryRow(ctx, `
		SELECT rs.id::text, rs.bucket, rs.object_key
		FROM reference_streams rs JOIN reference_actions ra ON ra.id = rs.action_id
		WHERE ra.sport=$1 AND ra.action=$2 AND rs.is_active=true
		ORDER BY rs.version DESC LIMIT 1`, sport, action).Scan(&r.ID, &r.Bucket, &r.Key)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// CreateSessionWithJob — 세션 + subjects(N) + 잡을 한 트랜잭션으로 기록. 반환: jobID.
// 스트림 원본은 이미 오브젝트 스토리지에 저장된 뒤 호출(메타는 성공 후에만 커밋).
func (s *Store) CreateSessionWithJob(ctx context.Context, st *contract.Stream, userID, referenceID,
	idempotencyKey, streamBucket, streamKey string, streamBytes int, handedness string) (string, error) {

	cp := st.Capture
	topo := cp.KeypointTopology
	if topo == "" {
		topo = "blazepose_33"
	}
	zsign := cp.ZSignConvention
	if zsign == "" {
		zsign = "toward_camera_negative"
	}

	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx) // 커밋되면 no-op

	_, err = tx.Exec(ctx, `
		INSERT INTO sessions
		(id,user_id,status,schema_version,source,model,model_variant,model_version,
		 keypoint_topology,fps,variable_framerate,frame_count,duration_s,
		 coordinate_space,dimensions,z_sign_convention,stream_bucket,stream_object_key,stream_bytes,handedness)
		VALUES($1::uuid,$2::uuid,'queued',$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)`,
		st.SessionID, userID, st.SchemaVersion, cp.Source, cp.Model, cp.ModelVariant, cp.ModelVersion,
		topo, cp.FPS, cp.VariableFramerate, cp.FrameCount, cp.DurationS,
		cp.CoordinateSpace, cp.Dimensions, zsign, streamBucket, streamKey, streamBytes,
		nullIfEmpty(handedness))
	if err != nil {
		return "", fmt.Errorf("세션 insert: %w", err)
	}

	for _, sub := range st.Subjects {
		fc := len(sub.Frames)
		_, err = tx.Exec(ctx,
			`INSERT INTO subjects(id,session_id,subject_key,label,frame_count)
			 VALUES($1::uuid,$2::uuid,$3,$4,$5)`,
			uuid.NewString(), st.SessionID, sub.SubjectID, sub.Label, fc)
		if err != nil {
			return "", fmt.Errorf("subject insert: %w", err)
		}
	}

	jobID := uuid.NewString()
	_, err = tx.Exec(ctx, `
		INSERT INTO analysis_jobs(id,session_id,reference_id,idempotency_key,status)
		VALUES($1::uuid,$2::uuid,$3::uuid,$4,'queued')`,
		jobID, st.SessionID, referenceID, idempotencyKey)
	if err != nil {
		return "", fmt.Errorf("job insert: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return jobID, nil
}

// UpsertResult — 잡×인물 결과 기록(멱등: 기존 행 삭제 후 삽입). score_breakdown/feedback 은 원시 JSON 바이트.
func (s *Store) UpsertResult(ctx context.Context, jobID, subjectID string, overall, dtw float64,
	scoreBreakdown, feedback, comparison []byte) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx,
		`DELETE FROM analysis_results WHERE job_id=$1::uuid AND subject_id=$2::uuid`, jobID, subjectID); err != nil {
		return err
	}
	var comp any
	if len(comparison) > 0 {
		comp = string(comparison)
	}
	if _, err = tx.Exec(ctx, `
		INSERT INTO analysis_results
		(id,job_id,subject_id,overall_score,dtw_distance,score_breakdown,feedback,comparison)
		VALUES($1::uuid,$2::uuid,$3::uuid,$4,$5,$6::jsonb,$7::jsonb,$8::jsonb)`,
		uuid.NewString(), jobID, subjectID, overall, dtw, string(scoreBreakdown), string(feedback), comp); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) MarkJobSucceeded(ctx context.Context, jobID, sessionID string) error {
	b := &pgx.Batch{}
	b.Queue(`UPDATE analysis_jobs SET status='succeeded', error=NULL, finished_at=now() WHERE id=$1::uuid`, jobID)
	b.Queue(`UPDATE sessions SET status='succeeded' WHERE id=$1::uuid`, sessionID)
	b.Queue(`SELECT pg_notify('job_done', $1)`, jobID+":succeeded") // SSE 푸시용
	return s.Pool.SendBatch(ctx, b).Close()
}

func (s *Store) MarkJobFailed(ctx context.Context, jobID, sessionID, errMsg string) error {
	b := &pgx.Batch{}
	b.Queue(`UPDATE analysis_jobs SET status='failed', error=$2, finished_at=now() WHERE id=$1::uuid`, jobID, errMsg)
	b.Queue(`UPDATE sessions SET status='failed' WHERE id=$1::uuid`, sessionID)
	b.Queue(`SELECT pg_notify('job_done', $1)`, jobID+":failed")
	return s.Pool.SendBatch(ctx, b).Close()
}

// ─────────────────────────── 읽기 ───────────────────────────

func (s *Store) ListReferences(ctx context.Context, sport string) ([]ReferenceCatalog, error) {
	q := `SELECT ra.sport, ra.action, rs.version
	      FROM reference_streams rs JOIN reference_actions ra ON ra.id=rs.action_id
	      WHERE rs.is_active=true`
	args := []any{}
	if sport != "" {
		q += ` AND ra.sport=$1`
		args = append(args, sport)
	}
	q += ` ORDER BY ra.sport, ra.action`
	rows, err := s.Pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ReferenceCatalog
	for rows.Next() {
		var c ReferenceCatalog
		if err := rows.Scan(&c.Sport, &c.Action, &c.Version); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// SessionSummary — 기록 목록 한 줄.
type SessionSummary struct {
	SessionID    string   `json:"session_id"`
	JobID        *string  `json:"job_id"`
	CreatedAt    string   `json:"created_at"`
	Status       string   `json:"status"`
	Sport        *string  `json:"sport"`
	Action       *string  `json:"action"`
	OverallScore *float64 `json:"overall_score"`
}

// ListUserSessions — 로그인 유저의 지난 분석 목록(최신순). 점수는 결과가 있으면 채워진다.
func (s *Store) ListUserSessions(ctx context.Context, userID string) ([]SessionSummary, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT s.id::text, j.id::text, s.created_at::text, s.status,
		       ra.sport, ra.action, ar.overall_score
		FROM sessions s
		LEFT JOIN analysis_jobs j     ON j.session_id = s.id
		LEFT JOIN reference_streams rs ON rs.id = j.reference_id
		LEFT JOIN reference_actions ra ON ra.id = rs.action_id
		LEFT JOIN analysis_results ar  ON ar.job_id = j.id
		WHERE s.user_id = $1::uuid
		ORDER BY s.created_at DESC
		LIMIT 50`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SessionSummary
	for rows.Next() {
		var x SessionSummary
		if err := rows.Scan(&x.SessionID, &x.JobID, &x.CreatedAt, &x.Status,
			&x.Sport, &x.Action, &x.OverallScore); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

// GetSessionStreamLoc — 워커가 스트림 원본을 로드할 버킷/키 + 손잡이(없으면 "").
// 손잡이는 좌우 반전 정규화 판단에 쓴다 (internal/normalize).
func (s *Store) GetSessionStreamLoc(ctx context.Context, sessionID string) (bucket, key, handedness string, err error) {
	var h *string
	err = s.Pool.QueryRow(ctx,
		`SELECT stream_bucket, stream_object_key, handedness FROM sessions WHERE id=$1::uuid`, sessionID).
		Scan(&bucket, &key, &h)
	if h != nil {
		handedness = *h
	}
	return
}

// ReferencesForAction — 주어진 reference_id 와 같은 동작(action)의 모든 레퍼런스.
//
// **채점 경로에서는 쓰지 않는다.** 채점은 잡에 고정된 대표 레퍼런스 1개로만 한다
// (여러 개 중 최고점을 택하면 점수 인플레이션 + 지연 N배 — cmd/worker 주석 참조).
// 이 메서드는 오버레이 비교·레퍼런스 매칭 같은 별도 기능에서 쓴다.
func (s *Store) ReferencesForAction(ctx context.Context, referenceID string) ([]Reference, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id::text, bucket, object_key FROM reference_streams
		WHERE action_id = (SELECT action_id FROM reference_streams WHERE id=$1::uuid)
		ORDER BY version`, referenceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Reference
	for rows.Next() {
		var r Reference
		if err := rows.Scan(&r.ID, &r.Bucket, &r.Key); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// GetReferenceByID — 워커가 레퍼런스 원본을 로드할 버킷/키 + 손잡이.
func (s *Store) GetReferenceByID(ctx context.Context, id string) (*Reference, error) {
	r := Reference{ID: id}
	var h *string
	err := s.Pool.QueryRow(ctx,
		`SELECT bucket, object_key, handedness FROM reference_streams WHERE id=$1::uuid`, id).
		Scan(&r.Bucket, &r.Key, &h)
	if err != nil {
		return nil, err
	}
	if h != nil {
		r.Handedness = *h
	}
	return &r, nil
}

// SubjectsMap — subject_key → subjects.id.
func (s *Store) SubjectsMap(ctx context.Context, sessionID string) (map[string]string, error) {
	rows, err := s.Pool.Query(ctx,
		`SELECT id::text, subject_key FROM subjects WHERE session_id=$1::uuid`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := map[string]string{}
	for rows.Next() {
		var id, key string
		if err := rows.Scan(&id, &key); err != nil {
			return nil, err
		}
		m[key] = id
	}
	return m, rows.Err()
}

// JobStatus — 잡 상태 조회(폴링용). 없으면 (\"\", pgx.ErrNoRows).
func (s *Store) JobStatus(ctx context.Context, jobID string) (string, error) {
	var status string
	err := s.Pool.QueryRow(ctx,
		`SELECT status FROM analysis_jobs WHERE id=$1::uuid`, jobID).Scan(&status)
	return status, err
}

// GetResults — 잡의 인물별 결과(점수·피드백). score_breakdown/feedback 은 JSON 그대로.
func (s *Store) GetResults(ctx context.Context, jobID string) ([]map[string]any, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT s.subject_key, r.overall_score, r.dtw_distance, r.score_breakdown, r.feedback, r.comparison
		FROM analysis_results r JOIN subjects s ON s.id=r.subject_id
		WHERE r.job_id=$1::uuid ORDER BY s.subject_key`, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var subjectKey string
		var overall, dtw float64
		var breakdown, feedback, comparison []byte
		if err := rows.Scan(&subjectKey, &overall, &dtw, &breakdown, &feedback, &comparison); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{
			"subject_key":     subjectKey,
			"overall_score":   overall,
			"dtw_distance":    dtw,
			"score_breakdown": rawJSON(breakdown),
			"feedback":        rawJSON(feedback),
			"comparison":      rawJSON(comparison),
		})
	}
	return out, rows.Err()
}

// rawJSON — DB의 jsonb 바이트를 응답에 그대로 싣기 위한 래퍼.
type rawJSON []byte

func (r rawJSON) MarshalJSON() ([]byte, error) {
	if len(r) == 0 {
		return []byte("null"), nil
	}
	return r, nil
}

// ReferenceMeta — 레퍼런스의 출처·권리·분류. 권리 근거 없는 레퍼런스는 등록할 수 없다.
//
// 레퍼런스는 타인의 자세를 제품의 기준으로 쓰는 자산이므로, "이 데이터를 쓸 근거가 무엇인가"에
// 항상 답할 수 있어야 한다(투자·심사·제휴 협상에서 반드시 요구된다).
type ReferenceMeta struct {
	SourceKind   string // 필수: self_recorded | permission | cc_licensed | synthetic
	RightsBasis  string // 필수: 허락 증빙 링크 · 라이선스 URL · 촬영 메모
	ProviderName string // 제공자(코치·채널명) — 크레딧 표기용
	Attribution  string // 표기 의무 문구 (CC-BY 등)
	Handedness   string // right | left (빈 값 허용)
	SkillLevel   string
	CameraAngle  string
	Notes        string
}

var validSourceKinds = map[string]bool{
	"self_recorded": true, "permission": true, "cc_licensed": true, "synthetic": true,
}

// Validate — DB에 닿기 전에 거부한다(에러 메시지를 사람이 읽을 수 있게).
func (m ReferenceMeta) Validate() error {
	if !validSourceKinds[m.SourceKind] {
		return fmt.Errorf("source_kind 가 올바르지 않음: %q (self_recorded|permission|cc_licensed|synthetic)", m.SourceKind)
	}
	if strings.TrimSpace(m.RightsBasis) == "" {
		return errors.New("권리 근거(rights_basis) 필수 — 허락 증빙·라이선스·촬영 메모 중 하나를 남겨야 등록된다")
	}
	if m.Handedness != "" && m.Handedness != "right" && m.Handedness != "left" {
		return fmt.Errorf("handedness 는 right|left 만 허용: %q", m.Handedness)
	}
	return nil
}

// nullIfEmpty — 빈 문자열을 SQL NULL 로 (선택 필드).
func nullIfEmpty(v string) *string {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return &v
}

// RegisterReference — "정답 예시" 레퍼런스를 카탈로그(reference_actions)+reference_streams 에 등록하고
// 해당 동작의 활성 레퍼런스로 만든다(기존 활성은 해제). 원본 JSON은 호출 전 오브젝트 스토리지에 저장.
// 멱등: 같은 (action, version) 재실행 시 활성/위치/메타데이터를 갱신.
func (s *Store) RegisterReference(ctx context.Context, sport, action string, version int,
	st *contract.Stream, bucket, key string, meta ReferenceMeta) (string, error) {
	if err := meta.Validate(); err != nil {
		return "", err
	}
	cp := st.Capture
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	var actionID string
	err = tx.QueryRow(ctx,
		`SELECT id::text FROM reference_actions WHERE sport=$1 AND action=$2`, sport, action).Scan(&actionID)
	if errors.Is(err, pgx.ErrNoRows) {
		actionID = uuid.NewString()
		if _, err = tx.Exec(ctx,
			`INSERT INTO reference_actions(id,sport,action) VALUES($1::uuid,$2,$3)`,
			actionID, sport, action); err != nil {
			return "", err
		}
	} else if err != nil {
		return "", err
	}

	// 기존 활성 해제
	if _, err = tx.Exec(ctx,
		`UPDATE reference_streams SET is_active=false WHERE action_id=$1::uuid`, actionID); err != nil {
		return "", err
	}

	var refID string
	err = tx.QueryRow(ctx, `
		INSERT INTO reference_streams
		(id,action_id,version,is_active,bucket,object_key,schema_version,model,model_variant,coordinate_space,dimensions,
		 source_kind,rights_basis,provider_name,attribution,handedness,skill_level,camera_angle,notes)
		VALUES($1::uuid,$2::uuid,$3,true,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)
		ON CONFLICT (action_id, version) DO UPDATE
		SET is_active=true, bucket=EXCLUDED.bucket, object_key=EXCLUDED.object_key,
		    source_kind=EXCLUDED.source_kind, rights_basis=EXCLUDED.rights_basis,
		    provider_name=EXCLUDED.provider_name, attribution=EXCLUDED.attribution,
		    handedness=EXCLUDED.handedness, skill_level=EXCLUDED.skill_level,
		    camera_angle=EXCLUDED.camera_angle, notes=EXCLUDED.notes
		RETURNING id::text`,
		uuid.NewString(), actionID, version, bucket, key, st.SchemaVersion,
		cp.Model, cp.ModelVariant, cp.CoordinateSpace, cp.Dimensions,
		meta.SourceKind, meta.RightsBasis, nullIfEmpty(meta.ProviderName), nullIfEmpty(meta.Attribution),
		nullIfEmpty(meta.Handedness), nullIfEmpty(meta.SkillLevel), nullIfEmpty(meta.CameraAngle),
		nullIfEmpty(meta.Notes)).Scan(&refID)
	if err != nil {
		return "", err
	}
	if err = tx.Commit(ctx); err != nil {
		return "", err
	}
	return refID, nil
}
