# verification — 검증 루프 · 완료 판정

> 태그: **Procedure**. Harness의 심장. `컴파일 성공 ≠ 기능 완료` 를 시스템으로 강제한다.

## 루프

```
Fixture → 실행 → Invariant / Golden 검증 → 수정 → 재검증 → 완료
```

## 2계층 검증

| 계층 | 무엇 | 언제 | 도구 |
| --- | --- | --- | --- |
| **Tier 1 · 코드 레벨** | 유닛/계약, 머지 게이트 | 구현 루프·PR | Fixture · Invariant · Golden · `go test` |
| **Tier 2 · 시스템 레벨** | 실사용 회귀 안전망 | 배포 직후 | API/E2E 스모크 → Notion 리포트 (예정) |

## 현재 갖춰진 것 (Fact)

- **Invariant** — `internal/contract` 의 INV-1..8 (경계 방어, `validate.py` 이식).
- **Fixture** — `.harness/fixtures/` (유효/무효 골격 스트림 샘플).
- **Runner** — `.harness/runners/`:
  - `check.sh` : `go build` + `go vet` + `go test` + fixture 검증 (머지 게이트)
  - `validate-fixture.sh <file>` : 단일 fixture 를 계약으로 검증
  - `smoke.sh` : 기동된 API 대상 업로드→폴링→결과 스모크

## 아직 없는 것 (백로그 = 다음 목표)

- **Golden File** — 승인된 분석 결과(점수·타점·교정)의 기준 출력. `.harness/golden/`.
  엔진 결정성 확보 후 대표 fixture의 결과를 golden 으로 승인·고정한다.
- 패키지별 `go test` 유닛 (현재 테스트 파일 0개) — 특히 `contract`(INV별 케이스)부터.
- Tier 2 E2E + Notion 테스트 리포트 DB 연동.

## 완료(Done)의 정의

1. `check.sh` 통과 (build + vet + test + fixture 검증)
2. 동작 변경 시 Golden/Invariant 갱신·승인
3. 환류: `wiki/`(Fact/Intent) 또는 `skills/`(Procedure) 갱신

## 첫 번째 우선 작업 (Phase 1 · 검증 루프 먼저)

원문 플랜의 "Phase 1 · 검증 루프 먼저 ⭐" 를 up-physical 에 적용:
1. `contract` 패키지 INV-1..8 각각의 유닛 테스트 (유효 통과 / 각 위반 검출).
2. 대표 포핸드 fixture 확정 → 엔진 결과 Golden 승인 → 회귀 고정.
3. `check.sh` 를 CI(develop 머지 게이트)에 연결.
