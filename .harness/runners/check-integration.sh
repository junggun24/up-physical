#!/usr/bin/env bash
# check-integration.sh — 기동된 스택 대상 통합 검증 (Tier 2).
#
# check.sh(머지 게이트)는 인프라 없이 결정적으로 돌아야 하므로 분리한다.
# 여기서는 DB·스토리지·큐·엔진이 함께 있어야 드러나는 동작을 검증한다:
#   채점이 활성 레퍼런스를 따르는가 · 손잡이 정규화 · 멱등 업로드
#
# 전제: deploy/ 스택 기동 + 마이그레이션 + API·워커 실행 (RUN.md 참조)
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

BASE="${UPX_API_BASE:-http://localhost:8080}"

if [ -f deploy/.env ]; then
  set -a; . deploy/.env; set +a
fi

if ! curl -fsS -m 3 "$BASE/healthz" >/dev/null 2>&1; then
  echo "✖ API 응답 없음: $BASE" >&2
  echo "  스택·API·워커를 먼저 띄우세요 (RUN.md)" >&2
  exit 1
fi

printf '\n\033[1m── 통합 검증 (%s) ──\033[0m\n' "$BASE"
UPX_INTEGRATION=1 go test ./internal/integration/ -count=1 -v

printf '\n\033[1;32m✔ check-integration 통과\033[0m\n'
