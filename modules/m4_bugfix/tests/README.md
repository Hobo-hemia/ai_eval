# M4 测试床

本目录提供 `m4_bugfix` 的可执行测试链路样例，目标是验证被测模型输出的 `m4_result.go` 是否满足：

- 修复并发与重入死锁风险
- 阈值告警按天 exactly-once
- 告警失败可重试，且失败调用不提交账务累计
- `context.Canceled` 失败后可恢复重试
- 累加溢出防护
- 保持既有对外签名
- 使用 `gomock` + `testify/assert` 通过关键边界测试

## 目录说明

- `harness/`: 真实执行的测试文件（会与被测 `m4_result.go` 放到同一包运行）
- `run_full_chain.sh`: 一键编译/测试脚本，自动写入 `eval_records/<model_dir>/m4_bugfix/` 日志

## 执行方式

在仓库根目录执行：

```bash
bash modules/m4_bugfix/tests/run_full_chain.sh <model_dir>
```

执行后会更新：

- `eval_records/<model_dir>/m4_bugfix/m4_build.log`
- `eval_records/<model_dir>/m4_bugfix/m4_test.log`

## 约定

被测文件（`m4_result.go`）必须使用 `package result`，并实现以下符号：

- `type RiskNotifier interface { NotifyHighRisk(context.Context, string, int64) error }`
- `func NewSettlementService(notifier RiskNotifier) *SettlementService`
- `func (s *SettlementService) AddTransactions(ctx context.Context, day string, amounts []int64) (int64, error)`
