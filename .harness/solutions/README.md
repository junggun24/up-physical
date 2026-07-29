# solutions — 해결책 라이브러리 (검색 가능한 조직 지식)

한 번 푼 문제를 두 번 풀지 않는다. 비자명한 문제를 해결할 때마다 여기 1파일로 남긴다.
미래 세션·에이전트가 grep으로 발견한다 — **검색될 단어**(에러 메시지 원문 등)를 그대로 포함할 것.

파일명: `YYYY-MM-DD-<slug>.md`

## 형식 (YAML frontmatter + 본문)

```markdown
---
problem: 한 줄 요약
module: api | worker | contract | store | queue | analysis | auth | infra | app
symptoms: ["에러 메시지 원문", "관찰된 증상"]
root_cause: 한 줄
tags: [dtw, idempotency, pgx, ...]
date: YYYY-MM-DD
---

## 증상
(무엇이 어떻게 실패했나 — 로그/출력 원문)

## 근본 원인
(왜 — 증거 포함)

## 해결
(무엇을 바꿨나 — 커밋 링크)

## 재발 방지
(어떤 테스트/fixture/불변식으로 고정했나)
```

## 규칙

- 해결책 없는 항목 금지 (진행 중이면 티켓/보드에 있어야 함).
- `harness-feedback` 스킬이 작업 종료 시 여기 기록 여부를 점검한다.
- 같은 문제가 재발하면: 기존 파일을 갱신하고 재발 방지가 왜 실패했는지 기록.
