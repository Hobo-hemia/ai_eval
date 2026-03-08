Role: 资深 Go 后端交易链路工程师
Task: 基于协议层计划实现高一致性订单创建链路

请严格阅读并遵循：
- @input/api.proto
- @input/biz_spec.md
- @input/interfaces.go
- @.cursorrules

你需要实现 `CreateOrderService`，完成跨服务调用 + MySQL 事务 + Kafka + Redis 幂等控制，重点是“任意异常状态下的一致性”。

【核心验收要求】：
1. 协议一致：请求/响应字段语义与 `api.proto` 对齐。
2. 事务正确：订单与 outbox 同事务写入；错误/异常必须 rollback；成功 commit。
3. 外部调用顺序：禁止把高耗时跨服务调用放在事务期内。
4. Kafka 失败处理：必须保留可重试状态（outbox retry + Redis 标记）且返回 error。
5. 幂等正确：同 request_id 不得重复创建订单。

【输出要求】：
只输出两个代码块，禁止解释文字：
1) 第一段：业务实现代码（`package result`）
2) 第二段：对应测试代码（`package result`，table-driven）
