-- 0003_session_handedness — 사용자 스윙의 손잡이.
--
-- Why: 좌완 사용자를 우완 레퍼런스와 비교하면 점수가 "자세 차이"가 아니라 "손잡이 차이"를
-- 반영한다. 워커가 손잡이가 다를 때만 좌우 반전(internal/normalize)을 적용하려면 세션마다
-- 손잡이를 알아야 한다. 레퍼런스 쪽 손잡이는 0002 에서 이미 기록한다.
--
-- NULL 허용: 앱이 아직 보내지 않는 기존/구버전 요청은 정규화를 건너뛴다(안전한 기본값).

BEGIN;

ALTER TABLE sessions
    ADD COLUMN handedness text,
    ADD CONSTRAINT sessions_handedness_check
        CHECK (handedness IS NULL OR handedness IN ('right', 'left'));

COMMIT;
