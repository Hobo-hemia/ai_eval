package workflow

import (
	"fmt"
	"regexp"
	"strings"

	"ai_eval/internal/module"
)

func phase1PromptByModule(moduleID string) (string, error) {
	switch moduleID {
	case "m1_arch":
		return strings.TrimSpace(`
你现在在 m1_arch 模块评测中，请严格按以下文件执行，不要省略任何要求：
- @input/prd.md
- @input/api.proto
- @input/guidance.md
- @.cursorrules

硬性要求：
1) 你是后端协议架构师，需要基于产品语言 PRD 抽象并重构协议，而不是按功能点逐条罗列接口。
2) 结果必须是“完整的改造后 proto 文件”，且 package 升级到 v2。
3) 必须包含：已有接口改名、字段新增/删除、以及新增能力的协议抽象。
4) 新增 rpc 总数必须小于 5（鼓励聚合抽象）。
5) 只输出一个代码块：改造后的 proto 内容（不要解释文字）。
`), nil
	case "m2_biz":
		return strings.TrimSpace(`
你现在在 m2_biz 模块评测中，请严格按以下文件执行，不要省略任何要求：
- @input/api.proto
- @input/biz_spec.md
- @input/interfaces.go
- @input/guidance.md
- @.cursorrules

硬性要求：
1) 按协议层语义实现 CreateOrder 业务主链路，覆盖多依赖调用（Pricing/Inventory/MySQL/Kafka/Redis）。
2) 保持 interfaces.go 规定的对外签名与接口契约。
3) 严禁重复声明 interfaces.go 中已定义的结构体与接口，只实现服务逻辑。
4) 保证任意依赖失败时业务正确且数据一致（事务、幂等、补偿）。
5) 只输出两段代码块，不要输出解释文字：
   - 第一段：业务实现代码（package result）
   - 第二段：对应 _test.go 测试代码（package result）
`), nil
	case "m3_component":
		return strings.TrimSpace(`
你现在在 m3_component 模块评测中，请严格按以下文件执行，不要省略任何要求：
- @input/guidance.md
- @input/rate_limit_spec.md
- @.cursorrules

硬性要求：
1) 按需求实现高度抽象、高可复用的并发缓存组件（泛型 + 配置化 + 高并发安全）。
2) 必须保留契约中规定的对外函数签名与类型名。
3) 必须在关键实现处加注释：// BUGFIX: [根因与修复逻辑]（至少两处）。
4) 测试必须包含 table-driven，并至少提供一个 benchmark。
5) 只输出两段代码块，不要输出解释文字：
   - 第一段：组件实现代码（package result）
   - 第二段：对应 _test.go 测试代码（package result）
`), nil
	case "m4_bugfix":
		return strings.TrimSpace(`
你现在在 m4_bugfix 模块评测中，请严格按以下文件执行，不要省略任何要求：
- @input/guidance.md
- @input/bug_report.md
- @input/legacy_code.go
- @.cursorrules

硬性要求：
1) 按 TDD 思路修复 legacy_code.go 中缺陷。
2) 必须保留对外签名不变。
3) 必须在关键修复处加注释：// BUGFIX: [根因与修复逻辑]
4) 测试必须使用 table-driven + gomock + testify/assert。
5) 只输出两段代码块，不要输出解释文字：
   - 第一段：修复后的业务代码（package result）
   - 第二段：对应 _test.go 测试代码（package result）
`), nil
	default:
		return "", fmt.Errorf("unsupported module for phase1 prompt: %s", moduleID)
	}
}

func phase3PromptByModule(moduleID, modelDir, judgeModel string, runtime module.RuntimeMetrics) (string, error) {
	switch moduleID {
	case "m1_arch":
		return strings.TrimSpace(fmt.Sprintf(`
请作为 M1 裁判严格评分。你必须遵循以下规则文件：
- @modules/m1_arch/JUDGE_AGENT.md

评分输入材料：
- @eval_records/%[1]s/m1_arch/m1_result.proto
- @eval_records/%[1]s/m1_arch/m1_build.log
- @eval_records/%[1]s/m1_arch/m1_test.log

运行时长指标（必须在输出 JSON 中填写 runtime_metrics）：
- phase1_seconds = %[3].1f
- phase2_seconds = %[4].1f
- phase3_seconds = %[5].1f
- total_seconds = %[6].1f

输出要求（强制）：
1) 只输出 JSON，禁止 markdown 和额外说明
2) 按 100 分制给出总分与分项
3) 重点评估：协议抽象能力、字段演进正确性、接口收敛质量
4) 在 JSON 中填写：
   - "judge_model": "%[2]s"
   - "runtime_metrics": {"phase1_seconds": x, "phase2_seconds": y, "phase3_seconds": z, "total_seconds": t}
`, modelDir, judgeModel, runtime.Phase1Seconds, runtime.Phase2Seconds, runtime.Phase3Seconds, runtime.TotalSeconds)), nil
	case "m2_biz":
		return strings.TrimSpace(fmt.Sprintf(`
请作为 M2 裁判严格评分。你必须遵循以下规则文件：
- @modules/m2_biz/JUDGE_AGENT.md

评分输入材料：
- @eval_records/%[1]s/m2_biz/m2_result.go
- @eval_records/%[1]s/m2_biz/m2_build.log
- @eval_records/%[1]s/m2_biz/m2_test.log

运行时长指标（必须在输出 JSON 中填写 runtime_metrics）：
- phase1_seconds = %[3].1f
- phase2_seconds = %[4].1f
- phase3_seconds = %[5].1f
- total_seconds = %[6].1f

输出要求（强制）：
1) 只输出 JSON，禁止 markdown 和额外说明
2) 按 100 分制给出总分与分项
3) 必须检查事务一致性、Kafka 失败补偿与 ctx 透传
4) 在 JSON 中填写：
   - "judge_model": "%[2]s"
   - "runtime_metrics": {"phase1_seconds": x, "phase2_seconds": y, "phase3_seconds": z, "total_seconds": t}
`, modelDir, judgeModel, runtime.Phase1Seconds, runtime.Phase2Seconds, runtime.Phase3Seconds, runtime.TotalSeconds)), nil
	case "m3_component":
		return strings.TrimSpace(fmt.Sprintf(`
请作为 M3 裁判严格评分。你必须遵循以下规则文件：
- @templates/phase2_judge_prompt.md
- @modules/m3_component/JUDGE_AGENT.md

评分输入材料：
- @eval_records/%[1]s/m3_component/m3_result.go
- @eval_records/%[1]s/m3_component/m3_build.log
- @eval_records/%[1]s/m3_component/m3_test.log

运行时长指标（必须在输出 JSON 中填写 runtime_metrics）：
- phase1_seconds = %[3].1f
- phase2_seconds = %[4].1f
- phase3_seconds = %[5].1f
- total_seconds = %[6].1f

输出要求（强制）：
1) 只输出 JSON，禁止 markdown 和额外说明
2) 按 100 分制给出总分与分项
3) 必须检查是否存在 // BUGFIX: 注释，否则 D3 至少扣 8 分
4) 在 JSON 中填写：
   - "judge_model": "%[2]s"
   - "runtime_metrics": {"phase1_seconds": x, "phase2_seconds": y, "phase3_seconds": z, "total_seconds": t}
`, modelDir, judgeModel, runtime.Phase1Seconds, runtime.Phase2Seconds, runtime.Phase3Seconds, runtime.TotalSeconds)), nil
	case "m4_bugfix":
		return strings.TrimSpace(fmt.Sprintf(`
请作为 M4 裁判严格评分。你必须遵循以下规则文件：
- @templates/phase2_judge_prompt.md
- @modules/m4_bugfix/JUDGE_AGENT.md

评分输入材料：
- @eval_records/%[1]s/m4_bugfix/m4_result.go
- @eval_records/%[1]s/m4_bugfix/m4_build.log
- @eval_records/%[1]s/m4_bugfix/m4_test.log

运行时长指标（必须在输出 JSON 中填写 runtime_metrics）：
- phase1_seconds = %[3].1f
- phase2_seconds = %[4].1f
- phase3_seconds = %[5].1f
- total_seconds = %[6].1f

输出要求（强制）：
1) 只输出 JSON，禁止 markdown 和额外说明
2) 按 100 分制给出总分与分项
3) 必须检查是否存在 // BUGFIX: 注释，否则 D3 记 0
4) 在 JSON 中填写：
   - "judge_model": "%[2]s"
   - "runtime_metrics": {"phase1_seconds": x, "phase2_seconds": y, "phase3_seconds": z, "total_seconds": t}
`, modelDir, judgeModel, runtime.Phase1Seconds, runtime.Phase2Seconds, runtime.Phase3Seconds, runtime.TotalSeconds)), nil
	default:
		return "", fmt.Errorf("unsupported module for phase3 prompt: %s", moduleID)
	}
}

func extractCodeBlocks(output string) []string {
	re := regexp.MustCompile("(?s)```(?:[a-zA-Z0-9_+-]+)?\\n(.*?)```")
	matches := re.FindAllStringSubmatch(output, -1)
	blocks := make([]string, 0, len(matches))
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		blocks = append(blocks, strings.TrimSpace(m[1])+"\n")
	}
	return blocks
}
