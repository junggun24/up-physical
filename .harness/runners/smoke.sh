#!/usr/bin/env bash
# smoke.sh — 기동된 API 대상 스모크: 업로드 → 잡 폴링 → 결과 조회.
# 전제: API 가 $BASE (기본 localhost:8080) 에 떠 있고 ALLOW_DEV_AUTH=true,
#       레퍼런스 시드(tennis/forehand) 등록 완료 (cmd/seed · runtime.md 참조).
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
BASE="${BASE:-http://localhost:8080}"
FIXTURE="$ROOT/.harness/fixtures/valid-forehand-2d.json"

echo "── healthz"
curl -fsS "$BASE/healthz" && echo

echo "── 업로드 (POST /v1/sessions)"
resp=$(curl -fsS -X POST "$BASE/v1/sessions" \
  -H "Idempotency-Key: $(uuidgen)" \
  -H "X-User-Id: harness-smoke" \
  -H "Content-Type: application/json" \
  -d "{\"stream\": $(cat "$FIXTURE"), \"analysis\": {\"sport\":\"tennis\",\"action\":\"forehand\"}}")
echo "$resp"
job_id=$(echo "$resp" | python3 -c 'import sys,json; print(json.load(sys.stdin)["job_id"])')

echo "── 잡 폴링 (최대 30초)"
for _ in $(seq 1 30); do
  status=$(curl -fsS "$BASE/v1/jobs/$job_id" | python3 -c 'import sys,json; print(json.load(sys.stdin)["status"])')
  echo "  status=$status"
  case "$status" in
    done|succeeded|completed) break ;;
    failed|error) echo "✖ 잡 실패"; exit 1 ;;
  esac
  sleep 1
done

echo "── 결과 (GET /v1/jobs/$job_id/results)"
curl -fsS "$BASE/v1/jobs/$job_id/results"
echo
echo "✔ smoke 통과"
