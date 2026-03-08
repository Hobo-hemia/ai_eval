# AI Coding 后端能力评测工程

当前主流程已切到 **Cursor 自动评测链路**，核心命令是：

- `ai_eval init`：初始化候选模型目录（只建目录）
- `ai_eval run`：执行 `Init -> Phase1 -> Phase2 -> Phase3 -> Done` 全链路
- `ai_eval clear`：清理评测产物

## 目录

- `modules/`: 模块题目、规则、测试床
- `templates/`: 通用裁判提示词
- `eval_records/`: 评测结果目录（按模型隔离）
- `cmd/ai_eval/`: 统一命令（init/run/clear）
- `docs/`: 评测文档

## 快速开始（当前推荐）

```bash
cd /path/to/ai_eval
cursor-agent login

go install ./cmd/ai_eval

ai_eval init --models "gemini-3.1-pro,gpt-5.2,sonnet-4.5,gpt-5.3-codex-low-fast" --modules "m4"
ai_eval run --module m4 --model "gpt-5.3-codex-low-fast" --judge-model "gemini-3-flash"

# 一轮评测结束后一键清空评测产物
ai_eval clear
```

## 当前行为（以 m4 为例）

- 仅支持模块：`m4` / `m4_bugfix`
- 自动产物：
  - `eval_records/<model_dir>/m4_bugfix/m4_result.go`
  - `eval_records/<model_dir>/m4_bugfix/m4_build.log`
  - `eval_records/<model_dir>/m4_bugfix/m4_test.log`
  - `eval_records/<model_dir>/m4_bugfix/score.json`
- 自动清理中间产物，避免后续编译污染
- `score.json` 自动包含：
  - `judge_model`
  - `runtime_metrics.phase1_seconds`
  - `runtime_metrics.phase2_seconds`
  - `runtime_metrics.phase3_seconds`
  - `runtime_metrics.total_seconds`

可选模型见 `docs/模型清单.md`，M4 细节见 `docs/M4_模块测试全链路样例.md`。

## Legacy 说明

`cmd/evaluator`、`cmd/ai_eval_init`、`cmd/ai_eval_clear` 为历史/兼容入口，推荐统一使用 `cmd/ai_eval`。
