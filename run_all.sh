#!/bin/bash

# 设置遇到错误时退出
set -e

echo "=========================================="
echo "🚀 AI Eval - 5模型 x 4模块 全量深水区评测"
echo "=========================================="

# 核心配置
MODELS="gpt-5.3-codex,opus-4.5,gemini-3-pro,kimi-k2.5,composer-1"
MODULES="m1,m2,m3,m4"
JUDGE="opus-4.6"

# 1. 编译最新的评测执行器
echo "▶️ [1/4] 正在编译评测工具 (bin_ai_eval)..."
go build -o bin_ai_eval .

# 2. 清空历史数据
echo "▶️ [2/4] 正在清空历史评测工作区 (eval_records)..."
./bin_ai_eval clear || true
rm -rf eval_records/*

# 3. 初始化目录结构
echo "▶️ [3/4] 正在初始化评测目录..."
echo "  - Models: $MODELS"
echo "  - Modules: $MODULES"
./bin_ai_eval init --models "$MODELS" --modules "$MODULES"

# 4. 嵌套循环执行矩阵测试
echo "▶️ [4/4] 开始执行 5x4 矩阵评测..."
echo ""

# 将逗号分隔的字符串转为数组
IFS=',' read -ra MODEL_ARRAY <<< "$MODELS"
IFS=',' read -ra MODULE_ARRAY <<< "$MODULES"

# 暂时关闭 set -e，防止某个单项评测挂掉导致整个脚本退出
set +e 

for model in "${MODEL_ARRAY[@]}"; do
    for module in "${MODULE_ARRAY[@]}"; do
        echo "--------------------------------------------------------"
        echo "🏃 当前进度: 待测模型 = [$model] | 测试模块 = [$module]"
        echo "⚖️  裁判模型: [$JUDGE]"
        echo "--------------------------------------------------------"
        
        ./bin_ai_eval run --module "$module" --model "$model" --judge-model "$JUDGE"
        
        if [ $? -ne 0 ]; then
            echo "⚠️  警告: [$model] 在 [$module] 的评测执行中出现异常中断，将跳过并继续..."
        fi
        echo ""
    done
done

echo "=========================================="
echo "✅ 全量评测执行完毕！"
echo "📂 所有测试代码、编译日志及裁判打分 (score.json) 已落盘至 ./eval_records 目录下。"
echo "=========================================="
