-- 0002_reference_metadata — 레퍼런스의 출처·권리·분류 축 추가.
--
-- Why: 레퍼런스는 타인의 자세를 제품의 "정답지"로 쓰는 자산이다. 지금까지는 version/is_active
-- 뿐이어서 (1) 권리 근거를 남길 곳이 없고 (2) "유저에게 어울리는" 레퍼런스를 고를 축이 없었다.
-- 수집을 늘리기 전에 분류 축이 있어야 한다 — 먼저 모으면 구분 불가능한 덩어리가 된다.
--
-- rights_basis 는 NOT NULL 이다: 근거 없는 레퍼런스는 구조적으로 등록될 수 없다.

BEGIN;

ALTER TABLE reference_streams
    ADD COLUMN source_kind   text,   -- 취득 경로 (아래 CHECK 참조)
    ADD COLUMN rights_basis  text,   -- 권리 근거: 허락 증빙 링크 · 라이선스 URL · 촬영 메모
    ADD COLUMN provider_name text,   -- 제공자(코치·채널명) — 앱 크레딧 표기용
    ADD COLUMN attribution   text,   -- 표기 의무 문구 (CC-BY 등). 없으면 NULL
    ADD COLUMN handedness    text,   -- 'right' | 'left' — 미러 정규화 판단에 쓴다
    ADD COLUMN skill_level   text,   -- 'coach' | 'advanced' | 'intermediate' 등 자유 태그
    ADD COLUMN camera_angle  text,   -- 'side' | 'front' | 'diagonal' — 도메인 차이 추적
    ADD COLUMN notes         text;

-- 기존 행 백필: 개발용 합성 레퍼런스라 저작물이 아니다.
UPDATE reference_streams
   SET source_kind  = 'synthetic',
       rights_basis = '개발 스텁(합성 데이터) — 저작물 아님'
 WHERE source_kind IS NULL;

ALTER TABLE reference_streams
    ALTER COLUMN source_kind  SET NOT NULL,
    ALTER COLUMN rights_basis SET NOT NULL,
    ADD CONSTRAINT reference_streams_source_kind_check
        CHECK (source_kind IN ('self_recorded', 'permission', 'cc_licensed', 'synthetic')),
    ADD CONSTRAINT reference_streams_rights_basis_check
        CHECK (length(btrim(rights_basis)) > 0),
    ADD CONSTRAINT reference_streams_handedness_check
        CHECK (handedness IS NULL OR handedness IN ('right', 'left'));

COMMIT;
