#!/bin/bash

set -euo pipefail

echo "=========================================="
echo "🚀 AI Eval - 批量评测（模型并行）"
echo "=========================================="

# 核心配置（可通过环境变量覆盖）
MODELS="${MODELS:-gpt-5.3-codex,opus-4.5,gemini-3-pro,kimi-k2.5,composer-1}"
MODULES="${MODULES:-m1,m2,m3,m4}"
JUDGE="${JUDGE:-opus-4.6}"
PARALLEL_JOBS="${PARALLEL_JOBS:-3}"

echo "▶️ [1/5] 正在编译评测工具 (bin_ai_eval)..."
go build -o bin_ai_eval .

echo "▶️ [2/5] 正在清空历史评测工作区 (eval_records)..."
./bin_ai_eval clear || true

echo "▶️ [3/5] 正在初始化评测目录..."
echo "  - Models: $MODELS"
echo "  - Modules: $MODULES"
./bin_ai_eval init --models "$MODELS" --modules "$MODULES"

echo "▶️ [4/5] 启动并行评测（单模型走批量 --module all）..."
echo "  - Parallel jobs: $PARALLEL_JOBS"
echo ""

IFS=',' read -ra MODEL_ARRAY <<< "$MODELS"

run_one_model() {
  local model="$1"
  local log_file="eval_records/${model}/run_all.log"
  mkdir -p "eval_records/${model}"
  echo "--------------------------------------------------------"
  echo "🏃 启动模型: [$model]（模块: all | 裁判: $JUDGE）"
  echo "📄 日志文件: $log_file"
  echo "--------------------------------------------------------"
  if ./bin_ai_eval run --module all --model "$model" --judge-model "$JUDGE" >"$log_file" 2>&1; then
    echo "✅ 模型 [$model] 评测完成"
    return 0
  fi
  echo "⚠️  模型 [$model] 评测失败，请查看日志: $log_file"
  return 1
}

running_pids=()
running_models=()
failed_models=()

for model in "${MODEL_ARRAY[@]}"; do
  run_one_model "$model" &
  running_pids+=("$!")
  running_models+=("$model")

  while [ "$(jobs -rp | wc -l | tr -d ' ')" -ge "$PARALLEL_JOBS" ]; do
    sleep 1
  done
done

for i in "${!running_pids[@]}"; do
  pid="${running_pids[$i]}"
  model="${running_models[$i]}"
  if ! wait "$pid"; then
    failed_models+=("$model")
  fi
done

echo "▶️ [5/5] 生成 RESULT.md ..."
./bin_ai_eval result

echo "=========================================="
echo "✅ 全量评测执行完毕"
echo "📂 结果目录: ./eval_records"
if [ "${#failed_models[@]}" -gt 0 ]; then
  echo "⚠️  以下模型执行失败: ${failed_models[*]}"
else
  echo "🎉 所有模型执行成功"
fi
echo "=========================================="
