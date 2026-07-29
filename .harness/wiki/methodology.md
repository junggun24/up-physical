# methodology — 복리 엔지니어링 방법론

> 태그: **Procedure/Intent**. 출처: [superpowers](https://github.com/obra/superpowers) ·
> [Compound Engineering](https://every.to/guides/compound-engineering) · [gstack](https://github.com/garrytan/gstack)
> 에서 up-physical에 맞게 선별 적용.

## 중심 원칙 (Compound)

**각 작업은 다음 작업을 더 쉽게 만들어야 한다.**
코드베이스는 시간이 지날수록 이해하기 쉽고, 수정하기 쉽고, 신뢰하기 쉬워져야 한다.
가치는 "작성한 코드량"이 아니라 "해결한 문제 수 + 하네스에 축적된 자산"이다.

## 메인 루프 (모든 작업의 형태)

```
Plan(40%) → Work(10%) → Review(40%) → Compound(10%)
```

| 단계 | 무엇 | 우리 도구 |
| --- | --- | --- |
| **Plan** | 요구 이해 · 코드베이스 유사 패턴 조사 · 설계 · 완료조건 | `plan-ticket` → `write-plan` → `.harness/plans/` |
| **Work** | 격리 브랜치에서 구현, 테스트 먼저 | `up-developer` (TDD) |
| **Review** | 검증 게이트 + 다각 리뷰, P1/P2/P3 분류 | `qa-verify` · `security-review` · check.sh |
| **Compound** | 해결책·패턴·배움을 검색 가능하게 축적 | `harness-feedback` → `.harness/solutions/` · wiki |

> 시간 배분의 진실: **계획과 검토에 80%, 타이핑과 축적에 20%.** 계획이 코드보다 중요하다.

## 50/50 규칙

작업 시간의 50%는 기능, **50%는 시스템 개선**(하네스·리뷰 자동화·문서·fixture)에 쓴다.
90/10로 가면 기술 부채가 복리로 쌓인다. 주기 점검: `retro` 스킬.

## TDD (새 동작은 테스트 먼저)

```
RED (실패 테스트 작성) → GREEN (최소 구현) → REFACTOR (동작 불변 정리)
```

- 새 동작·버그픽스는 **실패하는 테스트/fixture를 먼저** 만든 뒤 구현한다.
- 예외: 순수 문서·설정. 예외를 쓰면 커밋 본문에 사유 한 줄.

## 체계적 디버깅 (추측 금지)

증상 보고 "아마 이거겠지" 수정 금지. `debug-systematic` 스킬의 4단계:
재현 고정 → 가설·이분 탐색 → 근본 원인 증거 확인 → 수정+회귀 고정+환류.

## 승인 전 3질문 (사람/감독자가 AI 산출물 검수 시)

1. "여기서 **가장 어려웠던 결정**이 뭐야?" → 트레이드오프 확인
2. "**거절한 대안**과 이유는?" → 탐색 폭 검증
3. "**가장 확신 없는 부분**이 뭐야?" → 약점 파악

## 도입 단계 (현재 위치와 방향)

Stage 3(계획 신뢰: 계획 승인 후 실행 중 손놓기, PR 수준 검토) 정착 →
Stage 4(결과만 설명, 에이전트가 계획) → Stage 5(병렬 클라우드 실행) 지향.
계획 승인은 **명시적**으로 한다 (침묵 ≠ 동의).
