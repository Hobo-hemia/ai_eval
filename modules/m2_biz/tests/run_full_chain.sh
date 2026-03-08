#!/usr/bin/env bash
set -euo pipefail

MODEL_ID="${1:-}"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
MODULE_ID="m2_biz"
RECORD_DIR="$ROOT_DIR/eval_records/$MODEL_ID/$MODULE_ID"
RESULT_FILE="$RECORD_DIR/m2_result.go"
BUILD_LOG="$RECORD_DIR/m2_build.log"
TEST_LOG="$RECORD_DIR/m2_test.log"
HARNESS_DIR="$ROOT_DIR/modules/m2_biz/tests/harness"
CONTRACT_TYPES="$ROOT_DIR/modules/m2_biz/input/interfaces.go"

if [[ -z "$MODEL_ID" ]]; then
  echo "usage: bash modules/m2_biz/tests/run_full_chain.sh <model_dir>" >&2
  exit 2
fi

if [[ ! -f "$RESULT_FILE" ]]; then
  echo "missing result file: $RESULT_FILE" >&2
  exit 1
fi

mkdir -p "$ROOT_DIR/.tmp"
TMP_DIR="$(mktemp -d "$ROOT_DIR/.tmp/ai_eval_m2_chain.XXXXXX")"
cleanup() {
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT

cp "$RESULT_FILE" "$TMP_DIR/m2_result.go"
cp "$HARNESS_DIR"/*.go "$TMP_DIR/"
cp "$CONTRACT_TYPES" "$TMP_DIR/m2_contract_types.go"

pushd "$ROOT_DIR" >/dev/null
go mod tidy >/dev/null 2>&1
popd >/dev/null

pushd "$TMP_DIR" >/dev/null
{
  echo "[build] go test -tags m2harness -c ."
  go test -tags m2harness -c .
} >"$BUILD_LOG" 2>&1

{
  echo "[test] go test -tags m2harness -race -v ."
  go test -tags m2harness -race -v .
} >"$TEST_LOG" 2>&1
popd >/dev/null

echo "build log: $BUILD_LOG"
echo "test log:  $TEST_LOG"
