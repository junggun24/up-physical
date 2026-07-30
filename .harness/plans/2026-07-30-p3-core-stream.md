# P3 1단계 — 골격 스트림 코어 모듈 (앱-서버 계약 선검증)

승인: 사용자 지시 2026-07-30 ("기획쪽에 P3부터 진행") — MVP 기획서 §8 다음 액션 2

## 목표 / 완료조건

Android 앱의 심장인 **골격 스트림 생성기**를 순수 Kotlin 모듈로 먼저 만들고,
산출 JSON이 **서버 경계(contract INV-1..8)를 실제로 통과**함을 기계 검증한다.

- [x] `app/` Gradle 프로젝트 + `:core-stream` 모듈 (kotlin-jvm — Android SDK 불필요)
- [x] `SkeletonStreamBuilder`: MediaPipe 33랜드마크 프레임 → 계약 준수 스트림 JSON
- [x] TDD 유닛테스트: 33kp/id 규칙, t 단조증가, 2d/3d z 규칙, frame_count, 직렬화 필드명
- [x] **교차 검증**: 모듈이 생성한 샘플 JSON → Go `cmd/validate` (INV-1..8) 통과
- [x] 러너 `check-app.sh` (gradle test + 샘플 생성 + Go 검증)

## 조사 (Fact)

- 이 머신: JDK 24 ✅ · gradle ✖(brew 설치) · **Android SDK/Studio ✖** → 카메라/UI 모듈은
  SDK 설치 후 (2단계). 코어를 JVM 모듈로 분리하면 지금 전체 검증 루프가 돈다.
- MediaPipe Pose Landmarker(Android) = BlazePose 33 landmarks, normalized x,y + z + visibility
  → 계약의 `blazepose_33`/`image_normalized`와 1:1 대응.
- 계약 요구(코드 기준): 33개 id 0..32 정확히(INV-4), t 강한 증가(INV-2), 3d면 z 필수(INV-7),
  frame_count=최대 frames 길이(INV-8), source=on_device, space=image_normalized.

## 설계

| 결정 | 근거 | 거절한 대안 |
| --- | --- | --- |
| `app/` 멀티모듈, `:core-stream`은 kotlin-jvm | SDK 없이 지금 테스트 가능, 앱 모듈은 나중에 의존만 추가 | 바로 Android 프로젝트 (이 머신에서 컴파일 불가 = 검증 불능) |
| kotlinx.serialization + snake_case 명시 | 계약 필드명 고정 | Gson/Moshi (멀티플랫폼·컴파일타임 안정성 열세) |
| 비단조 t 프레임은 **드롭+카운트** | 기기 카메라의 중복 타임스탬프 현실 대응, 업로드 실패 예방 | 예외 던지기 (촬영 전체 실패 — UX 악화) |
| 기본 2d (z 생략) | 서버 fixture 정합, image space z는 신뢰도 낮음 | 3d 기본 (INV-7 리스크) |

## 단계

1. gradle 설치 → `app/` 스캐폴드 (+wrapper)
2. RED: 테스트 작성 → GREEN: 모델+빌더 구현 → 리팩토링
3. 샘플 생성 태스크 + `check-app.sh` (gradle test → 샘플 → `cmd/validate`)
4. 문서·환류 → ship

## 리스크

- JDK 24 + 최신 Gradle/Kotlin 호환 (문제 시 toolchain 17 고정)
- MediaPipe 실제 타입과의 어댑터는 2단계에서 확정 — 코어 입력을 단순 값 타입으로 유지해 결합 차단

## 결과 (2026-07-30 실행 완료)

- TDD 10케이스 GREEN (RED 선확인) + 교차 검증 통과: Kotlin 생성 스트림 → Go INV-1..8 OK.
- check-app.sh 러너 신설. 2단계(Android SDK 설치 후): :app 모듈 — CameraX+MediaPipe 어댑터
  (`Landmark` 로 매핑), 촬영 가이드 UI, 업로드 클라이언트.
