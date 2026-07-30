-- 0002_reference_metadata 롤백.
BEGIN;

ALTER TABLE reference_streams
    DROP CONSTRAINT IF EXISTS reference_streams_source_kind_check,
    DROP CONSTRAINT IF EXISTS reference_streams_rights_basis_check,
    DROP CONSTRAINT IF EXISTS reference_streams_handedness_check;

ALTER TABLE reference_streams
    DROP COLUMN IF EXISTS source_kind,
    DROP COLUMN IF EXISTS rights_basis,
    DROP COLUMN IF EXISTS provider_name,
    DROP COLUMN IF EXISTS attribution,
    DROP COLUMN IF EXISTS handedness,
    DROP COLUMN IF EXISTS skill_level,
    DROP COLUMN IF EXISTS camera_angle,
    DROP COLUMN IF EXISTS notes;

COMMIT;
