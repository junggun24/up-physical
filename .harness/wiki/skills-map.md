# skills-map — 명령 팔레트 (스킬 ↔ 출처 ↔ 언제 쓰나)

> 태그: **Procedure**. 호출: 채팅에 `/<스킬이름>` 또는 자연어. 새 스킬 추가 시 이 표를 갱신한다 (docs-sync 점검 항목).

## 루프 단계별 팔레트

| 단계 | 스킬 | 언제 | 출처(이식) |
| --- | --- | --- | --- |
| **Plan** | `plan-ticket` | 티켓 작성 (4요소 + 경량 범위 도전) | 자체 + compound |
| | `ceo-review` | 기획·범위·존재 이유를 심문 | gstack office-hours + plan-ceo-review |
| | `write-plan` | 반나절↑ 작업 계획 문서 + 승인 | superpowers writing-plans |
| | `eng-review` | 계획의 기술 잠금 (LOCK/CONDITIONAL/REJECT) | gstack plan-eng-review |
| **Design** | `design-shotgun` | 중요 화면 시안 3~4개 발산→수렴 | gstack design-shotgun |
| | `design-review` | 시안·핸드오프 채점 + 인계 판정 | gstack plan-design-review + design-review |
| **Work** | `debug-systematic` | 버그·이상 동작 (추측 수정 금지) | superpowers systematic-debugging ≈ gstack investigate |
| **Review** | `review` | 코드 리뷰 — 버그 사냥, P1 즉시 수정 | gstack review |
| | `qa-verify` | 검증 게이트 (Tier1 check.sh / Tier2 smoke) | 자체 ≈ gstack qa-only |
| | `security-review` | 보안 게이트 (OWASP·STRIDE, P1/P2/P3) | 자체 + gstack cso |
| | `devex-review` | 온보딩·개발 경험 실측 | gstack devex-review |
| **Ship** | `ship` | 게이트→환류→커밋→푸시(→PR) 원커맨드 | gstack ship |
| | `docs-sync` | 문서를 현실에 맞춤 (Trust Rule) | gstack document-release |
| **Compound** | `harness-feedback` | 작업 마감 환류 (Done의 마지막 단계) | 자체 ≈ gstack learn |
| | `retro` | 주기 회고 (환류율·50/50 점검) | gstack retro |

## 의도적으로 이식하지 않은 것 (사유)

| gstack 명령 | 사유 |
| --- | --- |
| `/qa` (브라우저 QA) | 웹 UI 없음. Android 앱(P3) 후 E2E 백로그와 합류 |
| `/land-and-deploy` `/canary` | 배포 파이프라인 미구축. CI/CD 구축 시 도입 |
| `/benchmark` | 현 병목은 백엔드 성능이 아님. "처리시간 실측→UX 예산" 티켓에서 기기 기준으로 수행 |
| `/browse` `/pair-agent` | Claude Code 내장 브라우저·에이전트 도구로 대체 |
| `/codex` | 외부 모델 교차 검토 — 필요 시 검토 |
| `/careful` `/freeze` `/guard` | Claude Code 권한 모드 + AGENTS.md 정책으로 대체 |
| `/setup-gbrain` (세션 메모리 DB) | wiki + solutions/ grep으로 충분한 규모 |
| `/design-consultation` `/design-html` | 디자인 시스템 미착수. 디자인 시스템 티켓 때 재검토 |
| `/document-generate` (Diataxis) | wiki 구조가 이미 역할 분리됨. 규모 커지면 재검토 |
