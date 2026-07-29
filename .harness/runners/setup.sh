#!/usr/bin/env bash
# setup.sh — 클론 후 1회 실행: 커밋 규칙 훅 + 커밋 메시지 템플릿 활성화.
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

git config core.hooksPath .githooks
git config commit.template .gitmessage
chmod +x .githooks/*

echo "✔ core.hooksPath = .githooks (commit-msg 규칙 강제)"
echo "✔ commit.template = .gitmessage"
echo "규칙 문서: .harness/wiki/conventions-git.md"
