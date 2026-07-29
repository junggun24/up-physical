# domain — 골격 스트림 계약 · 불변식 · 채점

> 태그: **Fact/Intent**. 계약 원본(단일 진실원)은 이 저장소 밖 `contracts/skeleton-stream/{SPEC.md,schema,validate.py}`.
> Go 경계 검증(`internal/contract`)은 `validate.py` 의 규칙을 이식한 것이다.

## 골격 스트림이란

앱이 온디바이스 BlazePose로 추출한 관절 시계열. 서버 분석의 유일한 입력.

```
Stream
├─ schema_version, session_id, created_at
├─ capture { source, model, model_variant/version, fps,
│            coordinate_space, dimensions, keypoint_topology, frame_count?, duration_s? }
└─ subjects[] { subject_id, label?, frames[] { t, keypoints[] { id, x, y, z?, visibility } } }
```

허용 열거값 (코드 기준):
- `capture.source` ∈ {`on_device`, `tracking_camera`}
- `capture.coordinate_space` ∈ {`image_normalized`, `world_metric`, `root_relative`}
- `capture.dimensions` ∈ {`2d`, `3d`}
- 토폴로지 기대 개수: `blazepose_33` → 33, `coco_17` → 17
  (model=blazepose 이고 topology 미지정이면 `blazepose_33` 로 간주)

## 불변식 INV-1..8 (경계 방어)

| ID | 규칙 | 강도 |
| --- | --- | --- |
| INV-1 | `subject_id` 는 세션 내 유일 | 실패 |
| INV-2 | 각 subject 의 프레임 `t` 는 강하게 증가 | 실패 |
| INV-3 | 프레임 내 keypoint `id` 중복 없음 | 실패 |
| INV-4 | 토폴로지 완전성: id 가 `0..N-1` 정확히 일치 | 실패 |
| INV-5/6 | (경고성 규칙) | **경고**(실패 아님) |
| INV-7 | `dimensions=3d` 이면 모든 keypoint `z` 존재 | 실패 |
| INV-8 | `capture.frame_count`(존재 시) == 최대 frames 길이 | 실패 |

> 코드: `internal/contract/contract.go` 의 `Validate`. 스키마 레벨 필수값(schema_version,
> session_id, source/space/dimensions 열거, fps>0, subjects≥1)도 같은 함수에서 검사한다.

## 채점(분석) 개요 (Intent)

- 포핸드 MVP 기준으로 **검증 완료된 Python 엔진**이 사용자 스트림을 레퍼런스와 **DTW 정렬**해
  구간별로 점수·타점·교정 포인트를 산출한다.
- 차별화 기능: "이번에 고칠 단 하나"(피드백 우선순위화), 코치 오버레이 비교, 성장 추적.
- 레퍼런스는 `ref_key` 로 엔진이 캐시(같은 키는 두 번째부터 바이트 전송 생략).

## 관련 백로그 (Notion)

- 분석 엔진(포핸드) 오프라인 검증 / 분석 워커 서비스화(엔진 래핑)
- 온디바이스 BlazePose 통합 → 골격 스트림 생성
- DB 스키마(N명 구조) + 마이그레이션
