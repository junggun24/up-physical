# runtime — 빌드 · 실행 · 인프라 · 환경변수

> 태그: **Procedure/Fact**. 원본 절차는 루트 `RUN.md`. 여기서는 하네스 관점 요약.

## 툴체인

- **Go** (모듈 `go 1.22`; 개발 머신엔 최신 Go 설치돼 있으면 됨).
- **Python3 + numpy + `engine/`** — 워커 호스트에서 분석 엔진 실행에 필요 (이 저장소 밖).
- 인프라: **Postgres + MinIO** (`deploy/docker compose up -d`, 이 저장소 밖).

## 환경변수

`deploy/.env` (예: `cp deploy/.env.example deploy/.env`):
`DATABASE_URL`, `S3_ENDPOINT`, `S3_ACCESS_KEY`, `S3_SECRET_KEY`, `JWT_SECRET`,
`ENGINE_SCRIPT`, `PORT`(기본 8080), `ALLOW_DEV_AUTH`(dev 시 `true` → `X-User-Id` 허용).

## 빌드 & 실행

```bash
go mod tidy          # 의존성 해석 (네트워크 필요)
go build ./...       # 컴파일
go vet ./...         # 정적 점검

# API
set -a; source deploy/.env; set +a
go run ./cmd/api      # :8080

# 워커 (다른 터미널)
set -a; source deploy/.env; set +a
ENGINE_SCRIPT=../engine/run_analysis.py go run ./cmd/worker
```

## 사전 준비 (스모크 전)

1. `deploy/` 인프라 기동 (Postgres + MinIO)
2. 마이그레이션 적용: `db/migrations/0001_init.up.sql`
3. 레퍼런스 시드 등록(`cmd/seed`) — `references_streams` + MinIO `reference.json`

## 한계 / 주의 (Fact)

- 인증(JWT 검증)은 게이트웨이/관리형 IdP 담당(UP-12). API는 dev 모드에서 `X-User-Id` 를
  검증 없이 신뢰한다 → **prod에서 `ALLOW_DEV_AUTH` 금지**.
- OTel 관측성, CI 게이트, 레퍼런스 운영 시드 스크립트는 백로그.
