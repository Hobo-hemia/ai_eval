# TODO LIST（你还需要完成的工作）

> 当前以 `ai_eval_init` + `ai_eval` 为主流程，以下仅保留尚未完成事项。

## Phase 1：题目物料补齐

### M1 架构生成
- [ ] 补全 `modules/m1_arch/input/api.proto` 的真实协议定义（service、message、字段约束、可选 PGV 约束）。
- [ ] 在 `modules/m1_arch/tests/` 增加黑盒测试（编译、校验调用、分层与错误包装规范）。

### M2 业务实现
- [ ] 补全 `modules/m2_biz/input/biz_spec.md` 的真实业务流程与边界条件。
- [ ] 补全 `modules/m2_biz/input/interfaces.go` 的真实依赖接口（DB、Kafka、RPC）。
- [ ] 在 `modules/m2_biz/tests/` 增加事务、容错、Context 透传测试。

### M3 组件设计
- [ ] 补全 `modules/m3_component/input/rate_limit_spec.md`（吞吐目标、窗口定义、压测口径）。
- [ ] 在 `modules/m3_component/tests/` 增加并发压测、`-race` 校验、Lua 原子性验证。

### M4 缺陷修复
- [x] 补全 `modules/m4_bugfix/input/bug_report.md` 与 `legacy_code.go` 的真实缺陷样本。
- [x] 在 `modules/m4_bugfix/tests/` 增加可复现原 Bug 的边界用例，覆盖 TDD 闭环。

## Phase 2：规则校准

- [ ] 确认每个模块的 `JUDGE_AGENT.md` 评分细则与你预期一致（可微调权重）。
- [ ] 确认 `templates/phase2_judge_prompt.md` 与各模块评分规则完全一致。

## Phase 3：评测执行与数据沉淀

- [ ] 逐模型、逐模块执行评测，并将代码与日志写入 `eval_records/<model_dir>/<module>/`。
- [ ] 记录每轮 Token 消耗与执行时长（当前手工记录）。
- [ ] 汇总多轮结果，输出最终评测结论（质量、稳定性、成本）。
