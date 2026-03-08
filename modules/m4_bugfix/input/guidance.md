Role: 资深 Go 后端开发工程师 / 质量保证专家
Task: TDD 缺陷自愈与单测闭环

请阅读缺陷描述 @bug_report.md 以及存在 Bug 的源文件 @legacy_code.go。请严格遵循当前工作区 .cursorrules，按照 TDD 原则定位并修复缺陷。

【必须保持的对外契约】：
1. 保持以下声明不变（名称、签名、语义）：
   - `type RiskNotifier interface { NotifyHighRisk(ctx context.Context, day string, amount int64) error }`
   - `func NewSettlementService(notifier RiskNotifier) *SettlementService`
   - `func (s *SettlementService) AddTransactions(ctx context.Context, day string, amounts []int64) (int64, error)`
2. 不允许更改常量含义：`riskThreshold` 语义仍为风险告警阈值。
3. 允许在实现内部新增必要私有 helper，但禁止改动对外方法签名。

【核心验收要求】：
1. 缺陷定位与注释（强制规则）：在修改代码的具体位置，必须使用 `// BUGFIX: [缺陷原因简述]` 添加修复注释。
2. 代码修复：彻底修复 `legacy_code.go` 中的隐患，确保不引入新的并发死锁、重复告警或溢出问题。
3. 关键行为：必须保证阈值告警按天 **exactly-once**，并避免持锁调用 notifier。
4. 测试驱动：使用 gomock 与 testify，编写可覆盖并发与边界条件的单测，验证 Bug 已被修复。

【输出要求】：
请按顺序输出两段代码块：
1. 修复后的完整 Go 业务代码。
2. 配套 `_test.go` 测试代码。
除代码块外，严禁输出任何分析或废话。
