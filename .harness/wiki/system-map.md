# system-map — 아키텍처·데이터 흐름·진입점

> 태그: **Fact** (현재 코드 동작). 충돌 시 코드가 진실 → 이 문서를 고친다.

## 큰 그림

```
[앱: 온디바이스 BlazePose]
      │  골격 스트림(JSON)
      ▼
POST /v1/sessions ──▶ cmd/api  ── 검증(contract) ─▶ objstore(원본 저장)
      │ 202 job_id                              └─▶ store(세션/subjects/잡 기록)
      ▼                                                    │ 잡 enqueue
   폴링 GET /v1/jobs/{id}                                  ▼
   GET  /v1/jobs/{id}/results ◀── store ◀── cmd/worker ──┐ queue(Postgres SKIP LOCKED)
                                                          └─ internal/analysis ─▶ Python 엔진
```

## 진입점

- **`cmd/api/main.go`** — 인제스션 & 조회 API. `net/http` + Go 1.22 메서드 라우팅.
  - `POST /v1/sessions` (Idempotency-Key 필수): 계약 검증 → objstore 저장 → 세션/잡 기록 → 202
  - `GET /v1/jobs/{id}`, `/v1/jobs/{id}/results`, `/v1/jobs/{id}/events`(SSE, `events.go`)
  - `POST /v1/auth/signup|login`, `GET /v1/references`, `GET /v1/sessions`
- **`cmd/worker/main.go`** — 큐 소비 → `internal/analysis` 로 엔진 호출 → 결과 저장. graceful shutdown.
- **`cmd/seed/main.go`** — 레퍼런스 시드 등록 도구.

## 내부 모듈 (경계)

| 패키지 | 책임 | 핵심 |
| --- | --- | --- |
| `internal/contract` | 골격 스트림 계약 타입 + 경계 검증 | `Parse`, `Validate`(INV-1..8) |
| `internal/store` | PostgreSQL 데이터 계층 | pgx, 세션/잡/레퍼런스/유저 |
| `internal/objstore` | S3 호환 오브젝트 스토리지 | minio-go, `PutBytes` |
| `internal/queue` | 잡 큐 | Postgres `FOR UPDATE SKIP LOCKED` (별도 브로커 없음) |
| `internal/analysis` | 엔진 호출 경계 | 상주 Python 엔진과 파이프 통신, ref 캐시 |
| `internal/auth` | 인증 | JWT 발급/검증, 비밀번호 해시 |

## 설계 의도 (Intent)

- **큐를 Postgres로** — 초기 운영 단순화. 별도 브로커(NATS/Redis) 도입 전까지 `SKIP LOCKED`.
- **엔진은 subprocess/상주 프로세스 경계 뒤** — 검증된 Python 엔진(DTW 채점)을 재이식 없이 재사용.
  운영 핫패스 네이티브화는 이후 과제.
- **원본 스트림 보존** — 검증된 바이트를 그대로 objstore에 저장(재분석·회귀 대비).
