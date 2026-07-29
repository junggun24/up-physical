#!/bin/sh
# Claude Code PostToolUse(Bash) 훅 — git commit 실행 감지 시 Notion 동기화 지시를 주입.
# 입력: stdin으로 훅 JSON ({tool_input:{command:...}}). 출력: additionalContext JSON(해당 시).
# 큐(.harness/tmp/notion-sync-pending.log)는 .githooks/post-commit 이 채운다.

cmd=$(jq -r '.tool_input.command // ""' 2>/dev/null)
case "$cmd" in
  *"git commit"*) ;;
  *) exit 0 ;;
esac

root="$(git rev-parse --show-toplevel 2>/dev/null)" || exit 0
queue="$root/.harness/tmp/notion-sync-pending.log"
[ -s "$queue" ] || exit 0

pending=$(cat "$queue")
jq -n --arg p "$pending" '{
  hookSpecificOutput: {
    hookEventName: "PostToolUse",
    additionalContext: ("[하네스 훅] 커밋이 Notion에 아직 반영되지 않았다. 대기 큐:\n" + $p + "\n\n규칙(AGENTS.md·docs-sync): (1) Notion 환류 로그 페이지에 이 커밋(들) 요약을 1항목으로 추가·병합하라. (2) 기획서 마일스톤·직무 페이지에 영향이 있으면 함께 갱신하라. (3) 반영 후 반드시 큐 파일 .harness/tmp/notion-sync-pending.log 를 비워라(> 로 truncate). 이미 이번 턴에 반영했다면 큐만 비워라.")
  }
}'
