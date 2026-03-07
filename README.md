# AI Coding 后端能力评测工程

本工程用于评测不同 AI 模型在后端真实代码场景中的能力，覆盖架构生成、业务实现、组件设计、缺陷修复四个模块。

## 你还需要完成的工作（当前必须）

> 现在工程骨架已完成，以下内容需要你补充真实评测物料后才能正式开跑。

### M1 架构生成
- 补全 `modules/m1_arch/input/api.proto` 的真实协议定义（service、message、字段约束、可选 PGV 约束）。
- 在 `modules/m1_arch/tests/` 增加黑盒测试（编译、校验调用、分层与错误包装规范）。

### M2 业务实现
- 补全 `modules/m2_biz/input/biz_spec.md` 的真实业务流程与边界条件。
- 补全 `modules/m2_biz/input/interfaces.go` 的真实依赖接口（DB、Kafka、RPC）。
- 在 `modules/m2_biz/tests/` 增加事务、容错、Context 透传测试。

### M3 组件设计
- 补全 `modules/m3_component/input/rate_limit_spec.md`（吞吐目标、窗口定义、压测口径）。
- 在 `modules/m3_component/tests/` 增加并发压测、`-race` 校验、Lua 原子性验证。

### M4 缺陷修复
- 补全 `modules/m4_bugfix/input/bug_report.md` 与 `legacy_code.go` 的真实缺陷样本。
- 在 `modules/m4_bugfix/tests/` 增加可复现原 Bug 的边界用例，覆盖 TDD 闭环。

### 评测执行与治理
- 确认每个模块的 `JUDGE_AGENT.md` 评分细则与你预期一致（可继续微调权重）。
- 评测时记录每轮模型的 Token 消耗与执行时长（当前需手工记录，可后续自动化）。

## 目录说明

- `modules/`: 核心测试床（每个模块包含专属规则、输入物料、测试床、裁判标准）
- `eval_records/`: 各模型隔离的结果目录（代码产物、编译日志、测试日志、评分 JSON）
- `templates/`: 通用 Prompt 模板（Phase 2 裁判指令）
- `cmd/evaluator/`: Go CLI 入口
- `internal/eval/`: 目录初始化与产物模板工具库
- `docs/`: 落地方案文档

## 模块清单

- `m1_arch`: 架构生成（gRPC 服务端骨架）
- `m2_biz`: 业务实现（事务 + 中间件 + 容错）
- `m3_component`: 组件设计（Redis Lua 限流器）
- `m4_bugfix`: 缺陷修复（TDD + gomock + assert）

## 推荐执行流程

1. 执行初始化：`go run ./cmd/evaluator -phase prepare`
2. 为目标模型 + 模块初始化记录目录：
   - `go run ./cmd/evaluator -phase record -model gpt-5.3 -module m1_arch`
3. 准备题目物料（放到 `modules/<module>/input/`）
4. 在 Cursor 中单独打开 `modules/<module>/` 目录作为 Workspace
5. 选定被测模型，使用该模块的 `input/guidance.md` 作为 Phase 1 提示词执行生成
6. 将产物与日志写入 `eval_records/<model>/<module>/`：
   - 代码：`mX_result.go`
   - 编译日志：`mX_build.log`
   - 测试日志：`mX_test.log`
7. 新建裁判会话（建议固定裁判模型），加载：
   - `templates/phase2_judge_prompt.md`
   - `modules/<module>/JUDGE_AGENT.md`
   - `eval_records/<model>/<module>/` 下三份客观材料
8. 将裁判 JSON 结果写回 `eval_records/<model>/<module>/score.json`
9. 按模型与模块汇总分数、Token 消耗、耗时，形成最终评测结论

## 如何使用这个评测 Project（一步一步）

### 1) 环境准备

- 安装 Go 1.22+。
- 使用 Cursor 作为执行环境。

### 2) 初始化项目

```bash
go run ./cmd/evaluator -phase prepare
```

该命令会创建/补齐 `modules/`、`eval_records/`、`templates/` 等目录结构。

### 3) 初始化一次评测记录

```bash
go run ./cmd/evaluator -phase record -model gpt-5.3 -module m1_arch
```

支持模型：
- `gemini-3.1`
- `gpt-5.3`
- `claude-opus-4.6`
- `qwen-3.5`
- `kimi-2.5`

支持模块：
- `m1_arch`
- `m2_biz`
- `m3_component`
- `m4_bugfix`

### 4) 执行 Phase 1（被测模型生成代码）

- 打开目标模块目录（例如 `modules/m1_arch/`）。
- 将 `input/guidance.md` 发给被测模型。
- 让模型基于 `input/` 中题目物料生成代码。
- 运行编译与测试，把日志保存到对应 `eval_records/<model>/<module>/`。

### 5) 执行 Phase 2（裁判评分）

- 新会话切换到裁判模型。
- 投喂 `templates/phase2_judge_prompt.md` 与模块 `JUDGE_AGENT.md`。
- 同时提供 `mX_result.go`、`mX_build.log`、`mX_test.log`。
- 要求只输出 JSON，并写入 `score.json`。

### 6) 汇总结果

- 按「模型 × 模块」统计分数。
- 结合 Token 与耗时，输出综合表现（质量、稳定性、成本）。
- 建议至少重复 3 轮评测，观察波动范围后再得出结论。

## 快速命令

```bash
go run ./cmd/evaluator -phase prepare
go run ./cmd/evaluator -phase record -model gpt-5.3 -module m1_arch
```
