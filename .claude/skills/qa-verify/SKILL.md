---
name: qa-verify
description: 업 피지컬 검증 게이트 실행 절차. 코드 변경 후 완료 판정, PR 머지 전 게이트, 회귀 확인이 필요할 때 사용. check.sh 기반 Tier 1 검증과 smoke 기반 Tier 2 검증을 순서대로 수행한다.
---

# 검증 게이트 실행

## Tier 1 — 코드 레벨 (항상)

```bash
.harness/runners/check.sh
```

내부적으로: `go build ./...` → `go vet ./...` → `go test ./...` → fixture 계약 검증.

- 유효 fixture(`valid-*.json`)는 **통과**해야 하고, 무효 fixture(`invalid-*.json`)는
  **실패 검출**되어야 정상이다. check.sh가 이 기대를 자동 확인한다.
- 실패 시: 출력 그대로를 증거로 남기고, 원인 수정 후 재실행. 통과할 때까지 완료 아님.

## Tier 2 — 시스템 레벨 (인프라 기동 시)

전제: Postgres+MinIO 기동 + 마이그레이션 + 레퍼런스 시드 (`.harness/wiki/runtime.md`).

```bash
.harness/runners/smoke.sh   # 업로드(202) → 잡 폴링 → 결과 조회
```

## 계약 변경 시 추가 절차

1. `fixtures/gen_fixtures.py` 로 fixture 재생성 (결정적 생성 — diff가 의도 변경만 반영해야 함)
2. `internal/contract/contract_test.go` 에 새 규칙의 위반 케이스 추가
3. `check.sh` 재실행

## 보고 형식

```
결과: 통과 / 실패
실행: <명령들>
증거: <핵심 출력 요약 — 테스트 수, fixture 판정>
실패 시: 재현 절차 / 기대 / 실제 / 심각도
```

## 환류 (실패는 자산이다)

새 버그 발견 → 재현 fixture를 `.harness/fixtures/`에 추가 → 회귀 테스트로 고정 →
원인·해결을 `.harness/wiki/`에 기록. 이 3단계 없이 버그를 닫지 않는다.
