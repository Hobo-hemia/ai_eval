# AI Coding 后端能力评测工程

当前主流程已切到 **Cursor 自动评测链路**，核心命令是：

- `ai_eval init`：初始化候选模型目录（只建目录）
- `ai_eval run`：执行 `Init -> Phase1 -> Phase2 -> Phase3 -> Done` 全链路
- `ai_eval clear`：清理评测产物

## 目录

- `main.go`: 程序入口
- `cmd/`: 命令实现（init/run/clear）
- `internal/`: 核心逻辑（internal + module + workflow）
- `modules/`: 模块题目、规则、测试床
- `templates/`: 通用裁判提示词
- `eval_records/`: 评测结果目录（按模型隔离）
- `docs/`: 分层文档（design/sop/reference/planning）

## 文件结构

```text
ai_eval/
├── main.go                     # 项目入口
├── cmd/
│   ├── init.go                 # init 子命令
│   ├── run.go                  # run 子命令与路由
│   └── clear.go                # clear 子命令
├── internal/
│   ├── internal.go             # internal 统一入口（对 cmd 提供稳定 API）
│   ├── module/                 # 模块元数据与通用模型（目录命名、评分结构、文件命名）
│   └── workflow/               # 自动化评测工作流拆分（phase1/phase2/judge/prompt/isolation）
├── modules/
│   ├── m3_component/           # M3 题目、输入材料、harness、裁判规则
│   └── m4_bugfix/              # M4 题目、输入材料、harness、裁判规则
├── templates/                  # 通用裁判提示词模板
├── eval_records/               # 评测产物输出（按 model/module 分桶）
└── docs/
    ├── design/                 # 设计与架构说明
    ├── sop/                    # 模块执行 SOP
    ├── reference/              # 模型清单等参考
    └── planning/               # TODO 与规划
```

## 快速开始（当前推荐）

```bash
cd /path/to/ai_eval
cursor-agent login

go install .

ai_eval init --models "gpt-5.2-codex,sonnet-4.5,gpt-5.3-codex-low-fast" --modules "m3,m4"
ai_eval run --module m3 --model "gpt-5.2-codex" --judge-model "gemini-3-flash"
ai_eval run --module m4 --model "gpt-5.3-codex-low-fast" --judge-model "gemini-3-flash"

# 一轮评测结束后一键清空评测产物
ai_eval clear
```

## 当前行为（以 m3/m4 为例）

- 当前支持模块：`m3` / `m3_component`、`m4` / `m4_bugfix`
- 自动产物：
  - `eval_records/<model_dir>/<module>/m3_result.go|m4_result.go`
  - `eval_records/<model_dir>/<module>/m3_build.log|m4_build.log`
  - `eval_records/<model_dir>/<module>/m3_test.log|m4_test.log`
  - `eval_records/<model_dir>/<module>/score.json`
- 自动清理中间产物，避免后续编译污染
- `Phase1` 在隔离工作区执行，只暴露当前模块 `input/` 与 `.cursorrules`，不暴露 `tests/harness` 与裁判规则
- 代码产出后才写入 `eval_records/<model_dir>/<module>/`，随后才进入 Phase2/Phase3
- `score.json` 自动包含：
  - `judge_model`
  - `runtime_metrics.phase1_seconds`
  - `runtime_metrics.phase2_seconds`
  - `runtime_metrics.phase3_seconds`
  - `runtime_metrics.total_seconds`

可选模型见 `docs/reference/模型清单.md`，模块细节见：
- `docs/sop/M3_模块测试全链路样例.md`
- `docs/sop/M4_模块测试全链路样例.md`

## 命令说明

当前仅维护统一入口 `ai_eval`（由根目录 `main.go` 与 `cmd/*.go` 实现）。
