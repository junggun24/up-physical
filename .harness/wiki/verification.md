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

## 채점 경로 회귀 검증 (수동 절차 — 자동화 전까지)

레퍼런스가 늘어도 채점이 흔들리지 않는지 확인한다. 워커 통합 테스트 하네스가 없어
현재는 기동된 스택에서 수동으로 돈다 (2026-07-30 이 절차로 결함 수정을 확인했다):

1. 활성 레퍼런스 v1 상태에서 fixture 업로드 → 점수 기록
2. 일부러 **더 안 맞는** v2 를 시드 (활성이 v2 로 바뀜) → 같은 fixture 업로드
3. 기대: 점수가 **v2 기준으로 낮아진다**. v1 점수가 유지되면 = 최고점 채택 결함 재발
4. 기대: 워커 로그에 `ref <id>` 가 잡당 **하나만** 찍힌다 (엔진 호출 1회)
5. 끝나면 v1 을 재시드해 활성 복원

> 2026-07-30 실측: v1 활성 → 88.5 / v2(나쁜 매칭) 활성 → **26**. 결함이 있었다면 26이 아니라
> 88.5가 유지됐을 것이다. 엔진 호출도 잡당 1회 확인.

## 손잡이 정규화 회귀 검증 (수동 — 워커 통합 하네스 전까지)

우완 레퍼런스 활성 상태에서 세 번 업로드해 점수를 비교한다:

| 입력 | `analysis.handedness` | 기대 |
| --- | --- | --- |
| `valid-forehand-2d.json` | `right` | 기준선 점수 |
| `valid-forehand-2d-lefty.json` | (없음) | **낮음** — 손잡이 차이가 점수에 섞인 상태 |
| `valid-forehand-2d-lefty.json` | `left` | **기준선과 같음** — 정규화가 손잡이 차이를 제거 |

> 2026-07-30 실측: 88.5 / 42.9 / 88.5. 워커 로그에 `손잡이 정규화 적용 (user=left, ref=right)` 1행.

## 아직 없는 것 (백로그 = 다음 목표)

- **Golden File** — 승인된 분석 결과(점수·타점·교정)의 기준 출력. `.harness/golden/`.
  엔진 결정성 확보 후 대표 fixture의 결과를 golden 으로 승인·고정한다.
- 패키지별 `go test` 유닛 — 특히 `contract`(INV별 케이스)는 완료, 나머지 패키지 미착수.
- **워커 통합 테스트 하네스** — DB·objstore·엔진을 띄운 상태의 자동 검증. 이게 없어서
  위 "채점 경로 회귀 검증"이 수동 절차로 남아 있다. 우선순위 높음(결함 재발 감지 불가).
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
