# conventions-git — 커밋 규칙

> 태그: **Procedure**. 강제 장치: `.githooks/commit-msg` (클론 후 `.harness/runners/setup.sh` 1회 실행).
> 규칙을 바꾸면 이 문서 + 훅 정규식 + `AGENTS.md` 요약을 **같은 커밋**에서 함께 바꾼다.

## 형식 (Conventional Commits 변형)

```
<type>(<scope>)?: <제목 — 명령형, 72자 이내, 마침표 금지>
                                        ← 빈 줄 필수
<본문 — 선택. "무엇"보다 "왜". 한국어 권장>
                                        ← 빈 줄
<푸터 — 선택. Refs, BREAKING CHANGE, Co-Authored-By>
```

## type (9종)

| type | 용도 |
| --- | --- |
| `feat` | 사용자/시스템 관점의 새 기능 |
| `fix` | 버그 수정 |
| `docs` | 문서만 (코드 무변경) |
| `refactor` | 동작 불변 구조 개선 |
| `test` | 테스트 추가·수정만 |
| `perf` | 성능 개선 |
| `ci` | CI·배포 파이프라인 |
| `chore` | 빌드·의존성·잡무 |
| `harness` | 하네스 자산 (wiki·skills·fixtures·golden·runners·agents) |

## scope (선택, 소문자)

패키지·영역명: `api` `worker` `seed` `contract` `store` `queue` `objstore` `analysis` `auth` `wiki` `fixtures` …

## 규칙

1. **제목은 명령형·결과 중심** — "무엇이 되는가". (예: `feat(api): 세션 목록 페이지네이션 추가`)
2. **본문은 Why** — 코드 diff가 말해주지 못하는 이유·배경·트레이드오프만 쓴다.
3. **1커밋 = 1의도** — 기능과 리팩토링을 섞지 않는다. 하네스 환류는 작업 커밋에 포함하거나 `harness:` 로 분리.
4. **Breaking change** — type 뒤 `!` (예: `feat(contract)!: …`) + 푸터에 `BREAKING CHANGE: 설명`.
5. AI 에이전트가 만든 커밋은 푸터에 `Co-Authored-By: Claude <...>` 유지.
6. `Merge…`/`Revert…`/`fixup!`/`squash!` 는 git 자동 형식 그대로 허용.

## 예시

```
feat(worker): 잡 실패 시 지수 백오프 재시도

엔진 프로세스 재시작 직후 첫 잡이 간헐 실패(파이프 워밍업)해
즉시 재시도가 오히려 실패율을 높였다. 3회 지수 백오프로 완화.

Refs: UP-31
```

```
harness(fixtures): INV-7 위반(3d에서 z 누락) 재현 샘플 추가
```

## 강제 장치

- `.githooks/commit-msg` — 형식 위반 시 커밋 거부 (정규식 + 빈 줄 검사, UTF-8 문자 기준 72자).
- 로컬 활성화: `.harness/runners/setup.sh` (클론 후 1회 — `core.hooksPath` + 커밋 템플릿 등록).
- `check.sh` 는 훅 미설정 시 경고를 띄운다 (게이트 실패는 아님).

## 커밋 → Notion 자동 동기화

모든 커밋은 Notion(환류 로그 등)에 반영된다. 2중 구조:

1. `.githooks/post-commit` — 커밋을 큐(`.harness/tmp/notion-sync-pending.log`)에 기록.
   Claude 밖에서 한 커밋도 큐에 쌓인다.
2. `.claude/settings.json` PostToolUse 훅 → `.harness/runners/notion-sync-reminder.sh` —
   `git commit` 실행 감지 + 큐 비어있지 않으면 에이전트에게 동기화 지시 주입.
   에이전트는 환류 로그(+영향 페이지)를 갱신하고 **큐를 비운다**.

큐가 남아 있으면 다음 커밋 때 다시 지시가 뜬다 (유실 방지). 수동 처리: `docs-sync` 스킬.
