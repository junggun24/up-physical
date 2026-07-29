---
name: ship
description: 출하 절차 (gstack /ship 이식). 작업을 끝냈을 때 게이트→환류→커밋→푸시(→PR)를 한 번에 정리한다. "커밋해줘", "올려줘", "마무리해줘"에 사용.
---

# Ship — 게이트 통과 확인 후 출하

## 순서 (하나라도 실패하면 중단하고 보고)

1. **게이트** — `.harness/runners/check.sh` 전체 통과 (build·vet·test·fixture).
   보안 영향 변경(auth·경계·비밀·egress)이 있으면 `security-review` 선행.
2. **환류 점검** — `harness-feedback` 질문표: solutions/plans/wiki/fixture 갱신 또는
   "배움 없음" 사유 확정. 갱신분은 이번 커밋에 포함.
3. **문서 정합** — 변경이 RUN.md·wiki와 어긋나는지 30초 스캔 (`docs-sync` 요약판).
4. **커밋** — 커밋 규칙(`conventions-git.md`): type(scope) 제목 + Why 본문.
   1커밋 = 1의도 — 섞였으면 분리.
5. **푸시** — master 직행(현 단독 개발) 또는 브랜치면 push + PR 생성(gh, 본문에 증거 요약).
6. **보고** — 커밋 해시 · 게이트 증거 요약 · 환류 항목 · 남은 티켓.

## 규칙

- 게이트 실패 상태로 커밋 금지 (WIP 필요 시 브랜치 + 제목에 `wip` scope 명시).
- 푸시 후 Notion 반영이 필요한 변경(기획·마일스톤·환류 로그)이면 함께 갱신.
- prod 배포·릴리스 태깅은 이 스킬 범위 밖 (배포 파이프라인 구축 후 land-deploy로 분리 예정).
