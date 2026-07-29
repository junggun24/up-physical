#!/usr/bin/env bash
# check.sh — 머지 전 게이트: build → vet → test → fixture 계약 검증.
# "컴파일 성공 ≠ 기능 완료" — 최소한 계약 검증 루프까지 통과해야 완료다.
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

step() { printf '\n\033[1m── %s ──\033[0m\n' "$1"; }

step "1/4 go build"
go build ./...

step "2/4 go vet"
go vet ./...

step "3/4 go test"
go test ./...

step "4/4 fixture 계약 검증 (INV-1..8)"
go run ./cmd/validate .harness/fixtures/valid-forehand-2d.json
go run ./cmd/validate -expect-invalid \
  .harness/fixtures/invalid-inv2-time.json \
  .harness/fixtures/invalid-inv4-topology.json

printf '\n\033[1;32m✔ check 통과\033[0m — build·vet·test·fixture 모두 성공\n'
