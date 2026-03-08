# M2 业务实现题：协议驱动的订单创建全链路

## 背景

你需要基于 `api.proto` 实现 `CreateOrder` 命令处理链路。该链路涉及：

- 跨服务调用：`PricingClient`、`InventoryClient`
- MySQL 事务：订单与 outbox 事件落库
- Kafka 消息投递：发布订单创建事件
- Redis：幂等控制与失败重试标记

目标是在任意依赖异常下保证业务正确性与数据一致性。

## 必须实现的对外契约（不可改签名）

参考 `interfaces.go`，必须实现：

- `func NewCreateOrderService(...) *CreateOrderService`
- `func (s *CreateOrderService) HandleCreateOrder(ctx context.Context, req *CreateOrderRequest) (*CreateOrderResponse, error)`

## 强制业务规则

1. **协议层一致性**
   - 以 `api.proto` 为准实现请求/响应结构（字段语义一致）。

2. **调用链顺序（高优先级）**
   - 先做参数校验与幂等占位，再做定价与库存预占，再进入 DB 事务落库；
   - 严禁在 DB 事务持有期间执行高耗时跨服务调用。

3. **事务一致性**
   - 开启事务后，订单和 outbox 必须同事务提交；
   - 任意落库错误必须 rollback；
   - commit 失败也必须按失败路径处理并返回 error。

4. **Kafka 失败容错**
   - Kafka 发布失败时，不能丢失业务意图；
   - 必须把 outbox 标记为待重试，并记录 Redis 重试标记；
   - 该调用应返回 error（由上层重试/补偿），但数据库中的订单与 outbox 必须保持一致状态。

5. **幂等与可重试**
   - 同 `request_id` 重放时，不可重复创建订单；
   - 幂等占位失败或异常时，必须清理占位，避免“脏占位”导致永久失败。

6. **错误与上下文**
   - 所有外部调用必须透传 `ctx`；
   - 不可吞错，不可 `_ = err`。

## 验收场景（测试将覆盖）

- 正常链路：全依赖成功，返回 CREATED
- 参数非法：快速失败，不触发下游调用
- 定价失败 / 库存失败：不应进入事务
- 事务内失败：必须 rollback
- Kafka 失败：订单/outbox 已落库，outbox 被标记重试，返回 error
- 幂等重放：不重复写库，返回历史订单结果
