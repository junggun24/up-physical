# runtime — 빌드 · 실행 · 인프라 · 환경변수

> 태그: **Procedure/Fact**. 실행 절차 원본은 루트 `RUN.md` (2026-07-30부터 인프라 자산이 리포 안에 있음).

## 툴체인

- **Go** (모듈 `go 1.22`) · **Python3 + numpy** (엔진) · **Docker Compose** (인프라) · **psql**

## 인프라 (리포 내 자산)

| 자산 | 위치 | 비고 |
| --- | --- | --- |
| Postgres 16 | `deploy/docker-compose.yml` | 호스트 **5433** (5432는 로컬 PG가 점유) |
| MinIO | 〃 | 9000(API) / 9001(콘솔) |
| 스키마 | `db/migrations/0001_init.up.sql` | store 코드에서 역산 — 원본 확보 시 diff |
| 엔진 | `engine/dev_engine_server.py` | **개발 스텁** — Golden 승인 금지, 실엔진으로 교체 예정 |
| 레퍼런스 | `.harness/fixtures/reference-forehand-2d.json` | `cmd/seed` 로 등록 |

## 환경변수 (`deploy/.env` ← `.env.example` 복사)

`DATABASE_URL`(5433), `S3_ENDPOINT`, `S3_ACCESS_KEY`, `S3_SECRET_KEY`, `JWT_SECRET`,
`ENGINE_SCRIPT`(기본 dev 스텁), `ENGINE_PYTHON`(기본 python3), `PORT`(8080),
`ALLOW_DEV_AUTH`(dev만 true — `X-User-Id` 무검증 신뢰, **prod 금지**).

## 기동 순서 (상세 명령은 RUN.md)

1. `deploy/` compose up → 2. 마이그레이션 → 3. `.env` → 4. `cmd/seed` (레퍼런스) →
5. `cmd/api` + `cmd/worker` → 6. `smoke.sh` 로 확인.

## 검증된 사실 (2026-07-30 P2 마감)

- E2E 스모크 통과: 업로드→계약검증→MinIO→큐→워커→엔진(스텁)→결과, 로컬 ~2초.
- 멱등성 확인: 같은 Idempotency-Key 2회 = 같은 job, attempts=1, 중복 분석 0.
- 알려진 P2 버그: `session_id` 재사용 업로드 → 500 db_error (4xx 매핑 필요) + 내부 에러 문자열 노출(보안 백로그 1번과 동일 뿌리).
