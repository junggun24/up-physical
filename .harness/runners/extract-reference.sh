#!/usr/bin/env bash
# extract-reference.sh — 영상 1개 → 레퍼런스 후보 골격 스트림 (추출 → 조립 → 계약 검증).
#
# 사용: .harness/runners/extract-reference.sh <영상> [시작초] [종료초]
# 출력: .harness/tmp/<이름>.landmarks.json, .harness/tmp/<이름>.stream.json
#
# 통과했다고 좋은 레퍼런스는 아니다 — 계약 통과는 최소 조건이고, "좋은 자세"는 사람이 승인한다.
# 시드(활성화)는 별도: go run ./cmd/seed -sport tennis -action forehand -version N -file <stream.json>
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

VIDEO="${1:?사용: extract-reference.sh <영상> [시작초] [종료초]}"
START="${2:-0}"
END="${3:-}"
NAME="$(basename "${VIDEO%.*}")"
LM=".harness/tmp/${NAME}.landmarks.json"
STREAM=".harness/tmp/${NAME}.stream.json"

step() { printf '\n\033[1m── %s ──\033[0m\n' "$1"; }
mkdir -p .harness/tmp

if [ ! -x .venv-pipeline/bin/python ]; then
  echo "✖ 파이프라인 환경 없음 — pipeline/README.md 의 설치 절차를 먼저 수행하세요" >&2
  exit 1
fi

step "1/3 랜드마크 추출 (MediaPipe)"
ARGS=(--start "$START")
[ -n "$END" ] && ARGS+=(--end "$END")
.venv-pipeline/bin/python pipeline/extract_pose.py "$VIDEO" "$LM" "${ARGS[@]}"

step "2/3 골격 스트림 조립 (:core-stream — 앱과 같은 빌더)"
(cd app && ./gradlew --console=plain -q :core-stream:buildStream --args="$LM $STREAM")

step "3/3 계약 검증 (INV-1..8)"
go run ./cmd/validate "$STREAM"

printf '\n\033[1;32m✔ 레퍼런스 후보 생성\033[0m %s\n' "$STREAM"
printf '  다음: 사람이 자세를 검수 → cmd/seed 로 활성화 (권리 근거 기록 필수)\n'
