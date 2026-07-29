# AGENTS.md — up-physical backend (작업 계약)

> 이 파일은 **상시 정책·작업 계약**이다. 짧게 유지한다. 상세 맥락은 `.harness/wiki/`,
> 절차는 `.harness/skills/`, 실행·검증은 `.harness/runners/` 로 위임한다.
> (Harness 프레임워크: [BuilderHub Agentic Development](https://app.notion.com/p/3a7b61e8ae4281cc9151eed3a00797d7))

## 1. 이 저장소가 무엇인가

`github.com/upphysical/backend` — 테니스 코칭(업 피지컬)의 **운영 백엔드**.
골격 스트림을 업로드받아 검증·저장하고(인제스션 API), 분석 워커가 큐를 소비해
Python 분석 엔진을 호출하고 결과를 저장한다.

```
cmd/api/      인제스션 & 결과 조회 API (net/http, Go 1.22 라우팅)
cmd/worker/   큐 소비 → 엔진 호출 → 결과 저장 (graceful shutdown)
cmd/seed/     레퍼런스 시드 도구
internal/contract/   계약 타입 + 불변식 검증(INV-1..8) — validate.py 이식
internal/store/      PostgreSQL 데이터 계층 (pgx)
internal/objstore/   S3 호환 스토리지 (minio-go)
internal/queue/      Postgres 잡 큐 (FOR UPDATE SKIP LOCKED)
internal/analysis/   Python 엔진 subprocess 호출 경계
internal/auth/        JWT 발급·검증, 비밀번호 해시
```

> **경계 주의** — 분석 엔진(`engine/`), 참조구현(`services/`), 계약 원본
> (`contracts/skeleton-stream/`), 인프라(`deploy/`), 마이그레이션(`db/`)은 **이 저장소 밖**에 있다.
> 여기서는 인터페이스 경계(`internal/analysis`, `internal/contract`)만 관리한다.

## 2. Trust Rule (충돌 시 진실의 우선순위)

**코드·테스트가 진실이다.** 문서(Wiki/Skill)와 코드가 어긋나면 코드를 믿고, 문서를 고친다.
지식은 3종으로 태깅한다: **Fact**(현재 동작) · **Intent**(설계/변경 이유) · **Procedure**(작업 방법).

## 3. 상시 정책 (지켜야 할 것)

- **Contract-first.** 골격 스트림은 경계에서 반드시 `contract.Parse` + `contract.Validate`(INV-1..8)로
  방어한다. 계약 규칙 변경은 `contracts/skeleton-stream/` 원본과 동기화되어야 한다.
- **멱등성.** `POST /v1/sessions` 는 `Idempotency-Key` 필수. 같은 키 = 같은 잡 반환.
- **부작용은 성공 후에만 커밋.** 스토리지 저장 성공 → 그 다음에 DB 메타 기록(순서 보존).
- **엔진 경계.** 분석 로직은 `internal/analysis` 뒤의 Python 엔진에만 둔다. 워커/Go 코드에
  점수 계산 로직을 복제하지 않는다.
- **컴파일 성공 ≠ 기능 완료.** 완료는 §5 의 검증 게이트를 통과해야 한다.

## 4. 명령 (러너)

| 명령 | 하는 일 |
| --- | --- |
| `.harness/runners/check.sh` | build → vet → test → fixture 검증 (머지 전 게이트) |
| `.harness/runners/validate-fixture.sh <file>` | 골격 스트림 fixture 를 계약(INV-1..8)으로 검증 |
| `.harness/runners/smoke.sh` | 기동된 API 대상 스모크(업로드→폴링→결과) |

## 5. 완료(Done)의 정의

PR 머지 조건:
1. `check.sh` 통과 (build + vet + test + fixture 검증)
2. 동작 변경 시 관련 **Golden/Invariant** 갱신 및 승인
3. 배운 것 환류 — `.harness/wiki/` (Fact/Intent) 또는 `.harness/skills/` (Procedure) 갱신

## 6. 더 읽기

- `.harness/wiki/system-map.md` — 아키텍처·데이터 흐름·진입점
- `.harness/wiki/domain/skeleton-stream.md` — 골격 스트림 계약·불변식·DTW
- `.harness/wiki/runtime.md` — 빌드·실행·인프라·환경변수
- `.harness/wiki/verification.md` — 검증 루프·완료 판정
