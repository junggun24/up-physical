-- 0003_session_handedness 롤백.
BEGIN;

ALTER TABLE sessions
    DROP CONSTRAINT IF EXISTS sessions_handedness_check,
    DROP COLUMN IF EXISTS handedness;

COMMIT;
