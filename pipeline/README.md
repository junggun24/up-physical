# pipeline — 영상 → 레퍼런스 골격 스트림

레퍼런스("정답지") 후보를 만드는 오프라인 파이프라인. 앱 카메라(F2) 완성과 무관하게 동작한다.

```
영상 ─▶ extract_pose.py ─▶ landmarks.json ─▶ :core-stream buildStream ─▶ stream.json
        (MediaPipe 33)                        (앱과 같은 빌더)            │
                                                                          ▼
                                                        cmd/validate (INV-1..8) ─▶ cmd/seed
```

**설계 원칙 — 계약 구현체를 늘리지 않는다.** Python 은 랜드마크 추출만 하고, 스트림 조립은
앱과 동일한 `SkeletonStreamBuilder`(Kotlin), 검증은 `cmd/validate`(Go) 하나로 통일한다.
계약 구현이 갈라지면 앱·서버·파이프라인이 서로 다른 데이터를 "유효"하다고 판단하게 된다.

## 설치 (1회)

```bash
python3 -m venv .venv-pipeline
.venv-pipeline/bin/pip install mediapipe opencv-python

mkdir -p pipeline/models
curl -L -o pipeline/models/pose_landmarker_full.task \
  "https://storage.googleapis.com/mediapipe-models/pose_landmarker/pose_landmarker_full/float16/latest/pose_landmarker_full.task"
```

`.venv-pipeline/` 과 `models/*.task`(9MB 바이너리)는 gitignore 대상이다.

## 사용

```bash
# 전체 흐름 (추출 → 조립 → 검증)
.harness/runners/extract-reference.sh coach_forehand.mp4

# 스윙 구간만 (편집 컷을 피해 좁힐 때)
.harness/runners/extract-reference.sh coach_forehand.mp4 12.0 14.5
```

## 품질 리포트 읽는 법

추출기가 내는 수치가 레퍼런스로 쓸 만한지 판정하는 근거다.

| 항목 | 의미 · 판단 |
| --- | --- |
| `인물검출` / `스캔` | 낮으면 인물이 잘렸거나 너무 멀다 |
| `토폴로지불완전` | 33개 미만 프레임. 계약 INV-4 를 만족 못 해 버려진다 |
| `채택률` | 실질 수율. 낮으면 구간·앵글을 바꾼다 |
| `평균 visibility` | 낮으면(<0.7) 가림·흐림이 많아 신뢰도가 낮다 |
| **`불연속(편집 컷 의심)`** | **한 스윙이 컷을 넘어가면 계약은 통과해도 쓸모없다.** 반드시 구간을 좁힌다 |

## 권리 근거는 필수다

레퍼런스는 타인의 자세를 제품의 기준으로 쓰는 자산이다. 등록 시 **권리 근거**(자체 촬영 /
명시적 허락 / CC 라이선스)를 함께 기록한다 — 근거 없는 레퍼런스는 등록하지 않는다.
허락받은 영상이라면 **원본 파일을 직접 받는 편이 낫다**: 플랫폼 재인코딩·편집 컷이 없어
수율과 품질이 올라가고, 플랫폼 약관 문제도 생기지 않는다.

## 한계 (2026-07-30)

- **실영상 검증 미완** — 조립·검증(후반부)과 실패 경로는 확인했지만, 실제 사람이 나오는
  영상으로 추출 품질을 측정한 적이 없다. 첫 실영상에서 채택률·visibility 기준선을 잡아야 한다.
- 단일 인물(`num_poses=1`) 고정. 코치+수강생이 함께 나오는 구간은 레퍼런스로 쓰지 않는다.
- `capture.source` 는 계약 열거값이 `{on_device, tracking_camera}` 뿐이라 영상 추출 소스를
  정확히 표기할 값이 없다 — 계약 소유자와 합의 필요 (현재는 앱과 동일 파이프라인 가정).
