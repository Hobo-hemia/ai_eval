Role: 顶级 Go 后端交易链路架构师
Task: 在极度恶劣的分布式网络环境下实现健壮的 CreateOrder 链路

请严格阅读并遵循：
- @input/api.proto
- @input/biz_spec.md
- @input/interfaces.go
- @.cursorrules

你需要实现 `CreateOrderService`。我们不再给你保姆级的步骤提示，请运用你的系统设计经验，自主解决重试风暴、接口非幂等、以及系统崩溃边缘的数据一致性问题。

【核心验收要求】：
1. 协议实现：精准对齐 `api.proto` 语义。
2. 并发防刷：有效拦截并发重放，坚决保护**非幂等下游**。
3. 性能保护：绝不可将网络 I/O 等慢操作包裹在数据库事务中引发 DBA 报警。
4. 极致一致性：当 Kafka 发生故障时，订单与消息事件的最终一致性必须得到保障。任何错误不能被默默吞掉。

【输出要求】：
只输出两个代码块，禁止解释文字：
1) 第一段：业务实现代码（`package result`）
2) 第二段：对应测试代码（`package result`，需使用 table-driven 测试极端异常场景）