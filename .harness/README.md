# .harness — 업 피지컬 통합 하네스

회사가 소유하는 **에이전트 개발 시스템**. 모델 능력을 *검증 가능한 결과*로 바꾼다.
**기획 · 디자인 · 개발 · QA · 보안 전 직무를 이 하나의 하네스로 관리한다.**
([Harness 프레임워크 원문](https://app.notion.com/p/3a7b61e8ae4281cc9151eed3a00797d7) ·
[에이전트 팀 패턴 참고: revfactory/harness](https://github.com/revfactory/harness/blob/main/README_KO.md))

## 4개 레이어 ↔ 이 디렉토리

| 레이어 | 역할 | 여기서 |
| --- | --- | --- |
| ① 지식·Context | 무엇을·어디·왜 | `wiki/` + 루트 `AGENTS.md` |
| ② 실행·도구 | 빌드/부분실행/디버깅 | `runners/` + `.claude/agents/`(직무 에이전트) + `.claude/skills/`(절차) |
| ③ 검증·완료판정 | 무엇이 맞고 끝인가 | `fixtures/` + `golden/` + `wiki/verification.md` |
| ④ 피드백·환류 | 실패·배움 축적 | `harness-feedback` 스킬 → `wiki/`·`skills/` 갱신 |

## 직무 매핑 (감독자 패턴)

```
up-supervisor (분배·증거 검수)
 ├─ up-planner   기획  → plan-ticket 스킬,  wiki/planning.md
 ├─ up-designer  디자인 → Figma MCP,        wiki/design.md
 ├─ up-developer 개발  → runners/check.sh,  wiki/system-map.md·runtime.md
 ├─ up-qa        QA    → qa-verify 스킬,    wiki/verification.md
 └─ up-security  보안  → security-review,   wiki/security.md
        (전원 마지막 단계: harness-feedback — 환류 없이 Done 없음)
```

## 구조

```
.harness/
├─ wiki/                      # AI Wiki (장기기억, 검색용으로 분할)
│  ├─ system-map.md           #   아키텍처·모듈·진입점
│  ├─ domain/skeleton-stream.md  # 골격 스트림 계약·불변식·DTW 도메인
│  ├─ runtime.md              #   빌드·실행·인프라·환경변수
│  ├─ verification.md         #   검증 루프·완료조건 (QA 기준)
│  ├─ planning.md             #   제품 방향·로드맵·티켓 규칙 (기획 기준)
│  ├─ design.md               #   디자인 원칙·핵심 화면·핸드오프 (디자인 기준)
│  └─ security.md             #   위협 모델·보안 규칙 (보안 기준)
├─ skills/                    # (저장소 로컬 절차 메모 — 실행 스킬은 .claude/skills/)
├─ fixtures/                  # 최소 입력(골격 스트림 샘플, 결정적 생성)
├─ golden/                    # 승인된 기준 출력
├─ runners/                   # check/validate/smoke CLI 래퍼
└─ e2e/                       # 배포후 E2E + Notion 리포터 (예정)

.claude/
├─ agents/                    # 직무 에이전트 6종 (supervisor/planner/designer/developer/qa/security)
└─ skills/                    # 실행 절차 4종 (plan-ticket/qa-verify/security-review/harness-feedback)
```

## 검증 루프 (Harness의 심장)

```
Fixture → 실행 → Invariant/Golden 검증 → 수정 → 재검증 → 완료
```

`컴파일 성공 ≠ 기능 완료`. 완료 조건이 있어야 Agent가 끝까지 루프를 돈다.
현 상태의 최소 루프: `runners/check.sh` (build+vet+test+fixture 검증).

## Notion 허브

[프로젝트 페이지](https://app.notion.com/p/3acb61e8ae42808781dccf9a907808da) 하위에
직무별 하네스 페이지(허브·기획·디자인·개발·QA·보안·환류 로그)가 있다.
리포와 Notion이 어긋나면 **리포(코드·이 문서)가 진실**이고 Notion을 고친다.
