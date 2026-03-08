#!/bin/bash

set -euo pipefail

MODELS="${MODELS:-gpt-5.3-codex,opus-4.5,gemini-3-pro,kimi-k2.5,composer-1}"
MODULES="${MODULES:-m1,m2,m3,m4}"
JUDGE="${JUDGE:-opus-4.6}"
PARALLEL_JOBS="${PARALLEL_JOBS:-3}"

echo "=========================================="
echo "AI Eval batch runner (parallel by model)"
echo "=========================================="

echo "[1/5] build evaluator binary..."
go build -o bin_ai_eval .

echo "[2/5] clear eval_records..."
./bin_ai_eval clear || true

echo "[3/5] init records..."
echo "  models: ${MODELS}"
echo "  modules: ${MODULES}"
./bin_ai_eval init --models "${MODELS}" --modules "${MODULES}"

echo "[4/5] run evaluations in parallel..."
echo "  judge model: ${JUDGE}"
echo "  parallel jobs: ${PARALLEL_JOBS}"

IFS=',' read -ra MODEL_ARRAY <<< "${MODELS}"
running_pids=()
running_models=()
failed_models=()

run_one_model() {
  local model="$1"
  local log_file="eval_records/${model}/run_all.log"
  mkdir -p "eval_records/${model}"
  echo "----------------------------------------"
  echo "start model=${model}, module=all, judge=${JUDGE}"
  echo "log=${log_file}"
  if ./bin_ai_eval run --module all --model "${model}" --judge-model "${JUDGE}" >"${log_file}" 2>&1; then
    echo "done model=${model} status=success"
    return 0
  fi
  echo "done model=${model} status=failed"
  return 1
}

for model in "${MODEL_ARRAY[@]}"; do
  run_one_model "${model}" &
  running_pids+=("$!")
  running_models+=("${model}")
  while [ "$(jobs -rp | wc -l | tr -d ' ')" -ge "${PARALLEL_JOBS}" ]; do
    sleep 1
  done
done

for i in "${!running_pids[@]}"; do
  pid="${running_pids[$i]}"
  model="${running_models[$i]}"
  if ! wait "${pid}"; then
    failed_models+=("${model}")
  fi
done

echo "[5/5] generate RESULT.md ..."
./bin_ai_eval result

echo "=========================================="
echo "batch run finished"
echo "result dir: ./eval_records"
if [ "${#failed_models[@]}" -gt 0 ]; then
  echo "failed models: ${failed_models[*]}"
else
  echo "all models succeeded"
fi
echo "=========================================="
