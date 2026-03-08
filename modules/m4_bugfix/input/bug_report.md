# M4 缺陷报告：高并发结算服务隐蔽故障组

## 背景

结算服务负责按日累加交易金额，并在金额超过阈值时通过通知器发送风险告警。  
线上处于高并发场景（峰值 2k RPS），最近出现“偶发卡死 + 告警风暴 + 数据异常”的组合事故。

核心对象和方法如下（定义在 `legacy_code.go`）：

- `SettlementService`
- `func NewSettlementService(notifier RiskNotifier) *SettlementService`
- `func (s *SettlementService) AddTransactions(ctx context.Context, day string, amounts []int64) (int64, error)`
- `type RiskNotifier interface { NotifyHighRisk(ctx context.Context, day string, amount int64) error }`

## 缺陷现象

线上观察到四类问题：

1. **重入死锁风险（P0）**  
   某些 notifier 实现会在告警回调中再次调用结算服务（补记审计日志），服务出现卡死。

2. **告警风暴（P0）**  
   当当日金额已超过阈值后，每次入账都会重复发送告警，导致下游风控系统被打爆。

3. **整数溢出导致总额异常（P1）**  
   极端大额请求下总额出现负值或异常跳变，怀疑 `int64` 累加溢出未保护。

4. **高并发吞吐下降（P1）**  
   pprof 显示锁持有时间过长，热点在告警路径。

## 复现条件

1. **重入死锁复现**
   - 使用一个 notifier：`NotifyHighRisk()` 内部再次调用 `AddTransactions()`。
   - 调用一次可触发阈值的请求（如 `[60, 50]`）。
   - 请求长时间不返回。

2. **重复告警复现**
   - 第一次调用入账使总额达到阈值（例如 `[80, 30]`）。
   - 再调用任意正数入账（如 `[1]`）。
   - 预期：只告警一次；实际：重复告警。

3. **溢出复现**
   - 调用 `AddTransactions(ctx, day, []int64{math.MaxInt64, 1})`。
   - 预期：返回 overflow 错误；实际：总额异常。

## 预期行为

1. `AddTransactions` 在并发场景保持线程安全，且不因 notifier 重入导致死锁。
2. 金额累加必须做 `int64` 溢出保护，发现溢出要返回错误。
3. 对同一天，阈值告警必须是 **exactly-once**（首次跨阈值触发，后续不重复）。
4. 不能在持锁区执行慢 I/O（包括 notifier 调用）。
5. 返回值应为当日累计金额，且负数/零金额按非法输入处理。
6. 保持最小侵入修复，不改变对外函数签名。
7. 修复代码必须在关键修复处带有：
   - `// BUGFIX: [根因与修复逻辑]`

## 验收要点（供测试使用）

- 使用 table-driven tests。
- 必须使用 `go.uber.org/mock/gomock`。
- 必须使用 `github.com/stretchr/testify/assert`。
- 覆盖以下边界：
  - 并发累计正确性（不丢账）
  - 重入 notifier 无死锁
  - overflow 防护
  - 阈值告警 exactly-once
  - 非法金额（<=0）
