#!/usr/bin/env bash
set -euo pipefail

MODEL_ID="${1:-}"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
MODULE_ID="m3_component"
RECORD_DIR="$ROOT_DIR/eval_records/$MODEL_ID/$MODULE_ID"
RESULT_FILE="$RECORD_DIR/m3_result.go"
BUILD_LOG="$RECORD_DIR/m3_build.log"
TEST_LOG="$RECORD_DIR/m3_test.log"
HARNESS_DIR="$ROOT_DIR/modules/m3_component/tests/harness"

if [[ -z "$MODEL_ID" ]]; then
  echo "usage: bash modules/m3_component/tests/run_full_chain.sh <model_dir>" >&2
  exit 2
fi

if [[ ! -f "$RESULT_FILE" ]]; then
  echo "missing result file: $RESULT_FILE" >&2
  exit 1
fi

mkdir -p "$ROOT_DIR/.tmp"
TMP_DIR="$(mktemp -d "$ROOT_DIR/.tmp/ai_eval_m3_chain.XXXXXX")"
cleanup() {
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT

cp "$RESULT_FILE" "$TMP_DIR/m3_result.go"
cp "$HARNESS_DIR"/*.go "$TMP_DIR/"

pushd "$ROOT_DIR" >/dev/null
go mod tidy >/dev/null 2>&1
popd >/dev/null

pushd "$TMP_DIR" >/dev/null
{
  echo "[build] go test -tags m3harness -c ."
  go test -tags m3harness -c .
} >"$BUILD_LOG" 2>&1

{
  echo "[test] go test -tags m3harness -race -v ."
  go test -tags m3harness -race -v .
  echo
  echo "[bench] go test -tags m3harness -run '^$' -bench 'BenchmarkShardCacheGetHitParallel' -benchmem ."
  go test -tags m3harness -run '^$' -bench 'BenchmarkShardCacheGetHitParallel' -benchmem .
} >"$TEST_LOG" 2>&1
popd >/dev/null

echo "build log: $BUILD_LOG"
echo "test log:  $TEST_LOG"
