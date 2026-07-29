---
name: docs-sync
description: 문서 동기화 절차 (gstack /document-release 이식). 코드·구조 변경 후 RUN.md·wiki·AGENTS.md·Notion이 현실과 일치하는지 스캔하고 Trust Rule로 고칠 때 사용. "문서 업데이트해줘", 릴리스 전 점검에 사용.
---

# Docs Sync — 문서를 현실에 맞춘다

Trust Rule: **코드·테스트가 진실.** 문서가 다르면 문서를 고친다 (반대 방향 금지).

## 1. 스캔 (변경 diff 기준 — 전체 문서 재작성 금지)

| 변경이 있으면 | 확인할 문서 |
| --- | --- |
| 실행 방법·env·포트·인프라 | `RUN.md`, `wiki/runtime.md`, `deploy/.env.example` |
| 아키텍처·모듈·엔드포인트 | `wiki/system-map.md` |
| 계약·불변식 | `wiki/domain/skeleton-stream.md` + `contracts/` 원본 동기화 여부 |
| 정책·완료 정의·커밋 규칙 | `AGENTS.md`, `wiki/conventions-git.md` |
| 마일스톤·범위·게이트 | Notion 기획서 + 해당 직무 하네스 페이지 |
| 스킬·에이전트 추가/변경 | `AGENTS.md` §6 표, `.harness/README.md`, `wiki/skills-map.md` |

## 2. 검증 방법

- 문서의 **명령은 실제로 실행**해본다 (복붙 → 동작 확인). 실행 못 하는 건 표시.
- 문서의 경로·파일명은 `ls` 로 실존 확인.
- 어긋남 발견 시: 즉시 수정 + 어긋난 채 지낸 기간이 길면 환류 로그에 원인 기록.

## 3. 산출

```
스캔 범위: <diff/영역>
수정: <문서별 무엇을 왜>
실행 검증: <돌려본 명령들>
Notion 반영: <페이지 · 변경>
```

주기 실행: retro 때 + 릴리스/마일스톤 마감 때. ship 스킬 3단계가 요약판을 수행.
