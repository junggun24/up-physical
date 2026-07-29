#!/usr/bin/env bash
# validate-fixture.sh — 골격 스트림 파일을 실제 경계 코드(internal/contract)로 검증.
#   사용: .harness/runners/validate-fixture.sh [-expect-invalid] <stream.json> [...]
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"
exec go run ./cmd/validate "$@"
