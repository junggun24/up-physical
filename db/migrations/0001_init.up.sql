-- 0001_init.up.sql — up-physical 백엔드 초기 스키마.
--
-- 주의: 원본 마이그레이션(작성 머신의 db/)이 유실되어 internal/store·queue 코드에서
-- 역산했다 (.harness/plans/2026-07-30-p2-closeout.md). 원본 확보 시 diff 필수.
-- 강한 제약(FK/UNIQUE/CHECK)은 코드가 아니라 DB가 강제한다.

BEGIN;

CREATE TABLE users (
    id            uuid PRIMARY KEY,
    external_id   text NOT NULL UNIQUE,          -- 'local:'+email 또는 IdP sub / 'dev:'+id
    email         text,
    password_hash text,
    created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE reference_actions (
    id     uuid PRIMARY KEY,
    sport  text NOT NULL,
    action text NOT NULL,
    UNIQUE (sport, action)
);

CREATE TABLE reference_streams (
    id               uuid PRIMARY KEY,
    action_id        uuid NOT NULL REFERENCES reference_actions(id),
    version          int  NOT NULL,
    is_active        boolean NOT NULL DEFAULT false,
    bucket           text NOT NULL,
    object_key       text NOT NULL,
    schema_version   text NOT NULL,
    model            text NOT NULL,
    model_variant    text NOT NULL,
    coordinate_space text NOT NULL,
    dimensions       text NOT NULL,
    created_at       timestamptz NOT NULL DEFAULT now(),
    UNIQUE (action_id, version)
);

CREATE TABLE sessions (
    id                 uuid PRIMARY KEY,
    user_id            uuid NOT NULL REFERENCES users(id),
    status             text NOT NULL CHECK (status IN ('queued','processing','succeeded','failed')),
    schema_version     text NOT NULL,
    source             text NOT NULL,
    model              text NOT NULL,
    model_variant      text NOT NULL,
    model_version      text NOT NULL,
    keypoint_topology  text NOT NULL,
    fps                double precision NOT NULL,
    variable_framerate boolean NOT NULL DEFAULT false,
    frame_count        int,
    duration_s         double precision,
    coordinate_space   text NOT NULL,
    dimensions         text NOT NULL,
    z_sign_convention  text NOT NULL,
    stream_bucket      text NOT NULL,
    stream_object_key  text NOT NULL,
    stream_bytes       int NOT NULL,
    created_at         timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX sessions_user_created_idx ON sessions (user_id, created_at DESC);

CREATE TABLE subjects (
    id          uuid PRIMARY KEY,
    session_id  uuid NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    subject_key text NOT NULL,
    label       text,
    frame_count int NOT NULL,
    UNIQUE (session_id, subject_key)                 -- INV-1을 DB에서도 강제
);

CREATE TABLE analysis_jobs (
    id              uuid PRIMARY KEY,
    session_id      uuid NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    reference_id    uuid REFERENCES reference_streams(id),
    idempotency_key text NOT NULL UNIQUE,            -- 멱등성의 원천
    status          text NOT NULL CHECK (status IN ('queued','processing','succeeded','failed')),
    error           text,
    attempts        int NOT NULL DEFAULT 0,
    created_at      timestamptz NOT NULL DEFAULT now(),
    started_at      timestamptz,
    finished_at     timestamptz
);
CREATE INDEX analysis_jobs_queued_idx ON analysis_jobs (created_at) WHERE status = 'queued';

CREATE TABLE analysis_results (
    id              uuid PRIMARY KEY,
    job_id          uuid NOT NULL REFERENCES analysis_jobs(id) ON DELETE CASCADE,
    subject_id      uuid NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
    overall_score   double precision NOT NULL,
    dtw_distance    double precision NOT NULL,
    score_breakdown jsonb NOT NULL DEFAULT '{}'::jsonb,
    feedback        jsonb NOT NULL DEFAULT '[]'::jsonb,
    comparison      jsonb,
    created_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (job_id, subject_id)                      -- 잡×인물 결과는 1행 (UpsertResult 멱등)
);

COMMIT;
