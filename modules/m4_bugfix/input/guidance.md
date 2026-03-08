Role: 顶级 Go 缺陷排查与自愈专家
Task: 诊断并彻底治愈高并发结算服务的连环故障

请阅读故障现象报告 @bug_report.md 以及存在严重历史包袱的源文件 @legacy_code.go。请严格遵循当前工作区 .cursorrules，按照 TDD 原则定位并修复缺陷。

【约束与红线】：
1. 保持接口签名绝对静止：
   - `type RiskNotifier interface { NotifyHighRisk(ctx context.Context, day string, amount int64) error }`
   - `func NewSettlementService(notifier RiskNotifier) *SettlementService`
   - `func (s *SettlementService) AddTransactions(ctx context.Context, day string, amounts []int64) (int64, error)`
2. 保持原有常量含义。
3. 如果告警过程中触发了任何失败（包括网络超时），必须保证**账务与告警状态的一致性**，不能出现“钱已经加上去了，但告警永远不再发送”的“脏账”状态。

【核心验收要求】：
1. 我们不再明示你具体的“死锁点”或“并发冲突点”，请通过观察故障报告中的表象（如死锁、余额跳变、告警风暴），自行发现 `legacy_code` 中隐藏的所有隐患。
2. 缺陷定位标注：在修改代码的具体位置，强制使用 `// BUGFIX: [你发现的深层根因与修复逻辑]` 添加注释（至少识别出 4 处独立缺陷）。
3. 单元测试反击战：使用 `gomock` 与 `testify` 编写“极端刁钻”的单元测试。测试必须模拟：大数溢出、下游依赖被取消后恢复、下游通知器在回调时重入系统等恶劣情况。

【输出要求】：
请按顺序输出两段代码块：
1. 修复后的完整 Go 业务代码。
2. 配套 `_test.go` 测试代码。
除代码块外，严禁输出任何分析或废话。