# 백엔드 실행 가이드 (Go)

인제스션 API + 분석 워커. 잡 큐는 별도 브로커 없이 **Postgres(SKIP LOCKED)** 로 처리.
분석은 검증된 **Python 엔진**(`engine/run_analysis.py`)을 워커가 subprocess로 호출(운영 핫패스 네이티브화 전까지).

## 구성

```
backend/
├── cmd/api/main.go        인제스션 & 결과 조회 API (net/http, Go 1.22 라우팅)
├── cmd/worker/main.go     큐 소비 → 엔진 호출 → 결과 저장 (graceful shutdown)
└── internal/
    ├── contract/          계약 타입 + 불변식 검증(INV-1..8) — validate.py 이식
    ├── store/             PostgreSQL 데이터 계층 (pgx)
    ├── objstore/          S3 호환 스토리지 (minio-go)
    ├── queue/             Postgres 잡 큐 (FOR UPDATE SKIP LOCKED)
    └── analysis/          Python 엔진 subprocess 호출 경계
```

## 사전 준비

1. 인프라: `cd deploy && docker compose up -d` (Postgres + MinIO).
2. 마이그레이션 적용: `db/migrations/0001_init.up.sql` 를 Postgres에 실행
   (예: `psql "$DATABASE_URL" -f db/migrations/0001_init.up.sql`).
3. 환경변수: `cp deploy/.env.example deploy/.env` 후 값 확인
   (`DATABASE_URL`, `S3_ENDPOINT`, `S3_ACCESS_KEY`, `S3_SECRET_KEY`, `ENGINE_SCRIPT`, `PORT`).
4. 엔진 런타임: 워커 호스트에 `python3` + `numpy` + `engine/` 가 있어야 함.

## 빌드 & 실행

```bash
cd backend
go mod tidy          # transitive 의존성 해석 (이 단계는 네트워크 필요)
go build ./...       # 컴파일 검증

# API
set -a; source ../deploy/.env; set +a
go run ./cmd/api      # :8080

# 워커 (다른 터미널)
set -a; source ../deploy/.env; set +a
ENGINE_SCRIPT=../engine/run_analysis.py go run ./cmd/worker
```

## 스모크 테스트 (앱 없이)

```bash
# 1) 레퍼런스 시드는 별도 도구로 등록 필요(references_streams + MinIO에 reference.json).
#    참조구현 services/references.register_reference 흐름을 운영 시드 스크립트로 옮길 것(UP 백로그).

# 2) 업로드
curl -sX POST localhost:8080/v1/sessions \
  -H "Idempotency-Key: $(uuidgen)" -H "X-User-Id: dev-1" \
  -H "Content-Type: application/json" \
  -d '{"stream": <골격스트림 JSON>, "analysis": {"sport":"tennis","action":"forehand"}}'
# → {"session_id","job_id","status":"queued"}

# 3) 폴링 → 결과
curl -s localhost:8080/v1/jobs/<job_id>
curl -s localhost:8080/v1/jobs/<job_id>/results
```

## 참고 / 한계

- 이 코드는 참조구현(`services/*.py`, 5/5 테스트 통과)을 Go로 이식한 것이다. **컴파일·통합 테스트는 Go 툴체인이 있는 PC에서** 수행한다(작성 환경은 네트워크가 막혀 검증 불가).
- 인증(JWT 검증)은 게이트웨이/관리형 IdP 담당(UP-12). API는 현재 `X-User-Id` 또는 Bearer의 `sub`를 검증 없이 사용한다.
- 레퍼런스 시드 운영 도구, OTel 관측성, CI 게이트는 백로그(보드 참조).
