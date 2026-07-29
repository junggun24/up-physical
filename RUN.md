# 백엔드 실행 가이드 (Go)

인제스션 API + 분석 워커. 잡 큐는 별도 브로커 없이 **Postgres(SKIP LOCKED)** 로 처리.
분석 엔진은 상주 Python 프로세스(라인 JSON 프로토콜, `internal/analysis`).

> 현재 리포에는 **개발 스텁 엔진**(`engine/dev_engine_server.py`)만 있다 — 파이프라인 검증 전용.
> 검증된 실엔진(포핸드 오프라인 검증본) 확보 시 `ENGINE_SCRIPT` 만 교체한다. **스텁 결과로 Golden 승인 금지.**

## 구성

```
cmd/api/main.go        인제스션 & 결과 조회 API (net/http, Go 1.22 라우팅)
cmd/worker/main.go     큐 소비 → 엔진 호출 → 결과 저장 (graceful shutdown)
cmd/seed/main.go       레퍼런스 시드 등록
cmd/validate/main.go   골격 스트림 계약 검증 CLI (하네스 러너용)
internal/              contract(INV-1..8) · store(pgx) · objstore(minio) · queue · analysis · auth
deploy/                docker-compose (Postgres:5433 + MinIO:9000) + .env.example
db/migrations/         스키마 (0001_init.up.sql — store 코드에서 역산, 원본 확보 시 diff 필요)
engine/                dev_engine_server.py (개발 스텁)
```

## 처음 한 번

```bash
.harness/runners/setup.sh                 # 커밋 훅 + 템플릿
cd deploy && docker compose up -d --wait  # Postgres(5433) + MinIO(9000/9001)
cp .env.example .env && cd ..
psql "postgres://upx:upx@localhost:5433/upphysical" -f db/migrations/0001_init.up.sql
set -a; source deploy/.env; set +a
go run ./cmd/seed -sport tennis -action forehand -version 1 \
  -file .harness/fixtures/reference-forehand-2d.json
```

## 실행

```bash
set -a; source deploy/.env; set +a
go run ./cmd/api      # :8080

# 다른 터미널
set -a; source deploy/.env; set +a
go run ./cmd/worker   # ENGINE_SCRIPT=engine/dev_engine_server.py (.env 기본값)
```

## 검증

```bash
.harness/runners/check.sh    # 머지 게이트: build·vet·test·fixture
.harness/runners/smoke.sh    # E2E: 업로드 → 잡 폴링 → 결과 (API·워커 기동 상태에서)
```

## 참고 / 한계

- 인증: dev 모드(`ALLOW_DEV_AUTH=true`)는 `X-User-Id` 를 검증 없이 신뢰 — **prod 금지**.
  정식 인증은 `POST /v1/auth/signup|login` → Bearer JWT.
- 알려진 P2: 클라이언트 제공 `session_id` 재사용 시 500(db_error) — 4xx 매핑 필요 (백로그).
- OTel 관측성, CI 게이트, Golden 승인(실엔진 필요)은 백로그 (`.harness/plans/2026-07-30-p2-closeout.md`).
