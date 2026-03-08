# AI Coding 后端能力评测工程

当前主流程已切到 **Cursor 自动评测链路**，核心命令是：

- `ai_eval_init`：初始化候选模型目录（只建目录）
- `ai_eval`：执行 `Init -> Phase1 -> Phase2 -> Phase3 -> Done` 全链路

## 目录

- `modules/`: 模块题目、规则、测试床
- `templates/`: 通用裁判提示词
- `eval_records/`: 评测结果目录（按模型隔离）
- `cmd/ai_eval_init/`: 初始化目录命令
- `cmd/ai_eval/`: 一键自动评测命令
- `docs/`: 评测文档

## 快速开始（当前推荐）

```bash
cd /Users/zhengyu.hu/Desktop/ai_eval
cursor-agent login

go install ./cmd/ai_eval_init
go install ./cmd/ai_eval

ai_eval_init --models "gemini-3.1-pro,gpt-5.3-codex,opus-4.6-thinking,kimi-k2.5,qwen3.5" --modules "m4"
ai_eval --module m4 --model "gpt-5.3-codex" --judge-model "opus-4.6"
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

`cmd/evaluator` 为历史骨架命令，当前自动化主流程不依赖它。
