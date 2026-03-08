#!/usr/bin/env bash
set -euo pipefail

MODEL_ID="${1:-}"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
MODULE_ID="m1_arch"
RECORD_DIR="$ROOT_DIR/eval_records/$MODEL_ID/$MODULE_ID"
RESULT_FILE="$RECORD_DIR/m1_result.proto"
BUILD_LOG="$RECORD_DIR/m1_build.log"
TEST_LOG="$RECORD_DIR/m1_test.log"
HARNESS_DIR="$ROOT_DIR/modules/m1_arch/tests/harness"

if [[ -z "$MODEL_ID" ]]; then
  echo "usage: bash modules/m1_arch/tests/run_full_chain.sh <model_dir>" >&2
  exit 2
fi

if [[ ! -f "$RESULT_FILE" ]]; then
  echo "missing result file: $RESULT_FILE" >&2
  exit 1
fi

mkdir -p "$ROOT_DIR/.tmp"
TMP_DIR="$(mktemp -d "$ROOT_DIR/.tmp/ai_eval_m1_chain.XXXXXX")"
cleanup() {
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT

cp "$RESULT_FILE" "$TMP_DIR/m1_result.proto"
cp "$HARNESS_DIR"/*.go "$TMP_DIR/"

pushd "$ROOT_DIR" >/dev/null
go mod tidy >/dev/null 2>&1
popd >/dev/null

pushd "$TMP_DIR" >/dev/null
{
  echo "[build] go test -tags m1harness -c ."
  go test -tags m1harness -c .
} >"$BUILD_LOG" 2>&1

{
  echo "[test] go test -tags m1harness -race -v ."
  go test -tags m1harness -race -v .
} >"$TEST_LOG" 2>&1
popd >/dev/null

echo "build log: $BUILD_LOG"
echo "test log:  $TEST_LOG"
