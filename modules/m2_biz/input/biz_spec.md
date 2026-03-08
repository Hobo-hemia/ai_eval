# M2 业务实现题：高并发订单创建与分布式协同

## 场景背景

你需要基于 `api.proto` 实现 `CreateOrder` 核心逻辑。这不仅是一个 CRUD，而是处在一个高并发、弱网络的真实微服务环境中。
你需要协同调度以下资源：
- 跨服务 RPC 调用：`PricingClient` (定价)、`InventoryClient` (库存)
- 数据库：MySQL 事务 (记录 Order 与 DomainEvent)
- 消息总线：Kafka (发布订单创建成功事件)

## 真实环境与“脏”挑战（必读）

你不是在理想的真空环境中写代码。请务必处理以下异常：
1. **网络超时与激进重试**：API 网关配置了激进的超时策略。同一个 `CreateOrder` 请求（拥有相同的 `request_id`）可能会在第 1 秒、第 3 秒被多次并发打到你的服务实例上。
2. **非幂等的底层接口**：坑爹的 `InventoryClient` 团队**没有实现幂等性**。如果你对同一个请求调用两次扣减库存，就会真的扣两次，导致超卖或业务报错！
3. **脆弱的下游总线**：Kafka 集群在高峰期偶尔会阻塞长达 5~10 秒，或者直接抛出 `Timeout`。如果 Kafka 挂了，你的订单已经入库，决不能因此丢失消息事件。
4. **数据库长事务预警**：DBA 监控脚本会直接 Kill 掉执行超过 1 秒的 MySQL 事务。

## 必须实现的对外契约（不可改签名）

参考 `interfaces.go`，必须实现：

- `func NewCreateOrderService(...) *CreateOrderService`
- `func (s *CreateOrderService) HandleCreateOrder(ctx context.Context, req *CreateOrderRequest) (*CreateOrderResponse, error)`

你可使用传入的 Redis 客户端（或其他机制）来防范并发安全陷阱。

## 验收场景（你的代码将被如何折磨？）

- 发起两次并发的相同 `request_id` 的请求。
- 模拟 `InventoryClient` 延迟，测试并发重入是否会导致双重扣减。
- 模拟 Kafka 100% 失败返回 error。
- 模拟 DB 提交前应用崩溃（断电容错性）。