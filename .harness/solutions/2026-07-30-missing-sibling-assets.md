---
problem: 리포가 참조하는 형제 자산(deploy/engine/db 마이그레이션)이 개발 머신에 없어 스택 기동 불가
module: infra
symptoms: ["ls ../deploy: No such file or directory", "환경변수 DATABASE_URL 필요", "RUN.md가 ../deploy/.env, ../engine/run_analysis.py, db/migrations/0001_init.up.sql 참조하나 파일 없음"]
root_cause: 원 작성 머신에만 있던 자산이 git 저장소 밖이라 클론에 딸려오지 않음
tags: [bootstrap, deploy, migration, engine-stub, docker]
date: 2026-07-30
---

## 증상

RUN.md·코드 주석이 `../deploy/`, `../engine/`, `db/migrations/` 를 참조하지만 클론에는 없음.
스택 기동·시드·스모크 전부 불가.

## 근본 원인

모노레포의 일부(backend/)만 git으로 옮겨져, 저장소 밖 형제 디렉토리 자산이 유실.

## 해결

코드에서 역산해 리포 안에 재구축 (커밋 참조: P2 마감 커밋):
- `deploy/docker-compose.yml` — Postgres 16(호스트 **5433** — 5432는 로컬 PG 점유) + MinIO
- `db/migrations/0001_init.up.sql` — `internal/store` 의 SQL에서 7테이블 역산 (CHECK/UNIQUE/FK 포함)
- `engine/dev_engine_server.py` — `internal/analysis/engine.go` 의 라인 프로토콜 구현 스텁
- `.harness/fixtures/reference-forehand-2d.json` — gen_fixtures.py 확장(위상 이동)으로 결정적 생성

## 재발 방지

- 실행에 필요한 인프라 자산은 전부 리포 안에 둔다 (클론 = 재현 가능).
- 역산 스키마임을 마이그레이션 파일 머리에 명시 — 원본 확보 시 diff 절차 남김.
- 남은 외부 의존은 실엔진 1개뿐임을 RUN.md에 명시.
