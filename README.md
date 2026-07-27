# backend (Go)

운영용 백엔드. `../services/` 의 Python 참조구현(인제스션·워커·오브젝트스토어, 5/5 테스트 통과)을
Go 로 이식한다. 계약 검증은 `../contracts/skeleton-stream/` 스키마를 따른다.

## 구조

```
backend/
├── cmd/
│   ├── api/      인제스션 API 서버 (업로드 검증·저장·큐 등록)
│   └── worker/   분석 워커 (정규화·DTW·채점·타점·리포트)
├── internal/
│   ├── contract/   계약 검증 (스키마 로드·불변식)
│   ├── store/      Postgres 접근
│   ├── objstore/   MinIO/S3 클라이언트
│   ├── queue/      잡 큐 (NATS / Redis Streams)
│   └── analysis/   분석 엔진 바인딩 (engine 이식 대상)
└── go.mod
```

## 참조 매핑 (Python → Go)

| Python (services/) | Go (backend/internal/) |
|---|---|
| validation.py | contract/ |
| db.py | store/ |
| objectstore.py | objstore/ |
| ingestion.py | cmd/api + internal |
| worker.py | cmd/worker + analysis/ |

## 로컬 실행 (예정)

```bash
go run ./cmd/api      # 인제스션 API
go run ./cmd/worker   # 워커
```

> 의존 인프라: `../deploy/docker-compose.yml` (Postgres + MinIO) 먼저 기동.
