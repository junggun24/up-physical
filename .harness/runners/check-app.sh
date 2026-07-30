#!/usr/bin/env bash
# check-app.sh — 앱 코어(:core-stream) 게이트: 유닛테스트 + 계약 교차 검증.
# 앱이 생성한 골격 스트림이 서버 경계(cmd/validate, INV-1..8)를 통과해야 완료다.
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

step() { printf '\n\033[1m── %s ──\033[0m\n' "$1"; }

step "1/2 앱 코어 유닛테스트 (gradle)"
(cd "$ROOT/app" && ./gradlew --console=plain -q :core-stream:test)

step "2/2 계약 교차 검증 (Kotlin 생성 → Go INV-1..8)"
mkdir -p "$ROOT/.harness/tmp"
(cd "$ROOT/app" && ./gradlew --console=plain -q :core-stream:run --args="$ROOT/.harness/tmp/app-sample-stream.json")
(cd "$ROOT" && go run ./cmd/validate .harness/tmp/app-sample-stream.json)

printf '\n\033[1;32m✔ check-app 통과\033[0m — 테스트 + 앱→서버 계약 정합\n'
