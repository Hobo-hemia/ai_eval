Role: 资深 Go 后端架构师
Task: 复杂业务流转与中间件调度实现

请仔细阅读需求文档 @biz_spec.md 以及依赖接口定义 @interfaces.go，严格遵循当前工作区 .cursorrules，实现该模块的核心业务逻辑。

【核心验收要求】：
1. 事务边界：必须包含严谨的 PolarDB/MySQL 事务控制，确保在发生 Error 或 Panic 时能够正确 Rollback，成功则 Commit。
2. 容错与重试：实现对 Kafka 或外部 RPC 的调用，并包含必要的错误降级（Fallback）或重试机制。
3. 上下文控制：所有外部调用必须正确透传 context.Context，并处理好超时（Timeout）或取消（Cancellation）边界。

【输出要求】：
请只输出包含核心业务逻辑的完整 Go 代码（包含必要的 Struct 和 Func）。无需解释设计思路，确保代码可直接参与 go build。
