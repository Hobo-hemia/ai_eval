# M2 测试床

用于验证 M2 业务模块是否具备“协议驱动 + 多依赖容错 + 一致性保障”能力。

## 覆盖目标

- `api.proto` 字段语义对应到业务对象
- 参数校验与幂等控制
- 跨服务调用错误路径（Pricing/Inventory）
- MySQL 事务正确性（CreateOrder + Outbox 原子提交）
- Kafka 失败下的 outbox retry 与 Redis 标记
- Context 透传与错误不吞掉

## 目录

- `harness/`: 合同测试
- `run_full_chain.sh`: 一键构建+测试（含 `-race`）
