package workflow

import (
	"fmt"
	"regexp"
	"slices"
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

func phase3PromptByModule(
	moduleID, modelDir, judgeModel string,
	runtime module.RuntimeMetrics,
	phase2Failed bool,
	phase2Error string,
) (string, error) {
	phase2Status := "pass"
	if phase2Failed {
		phase2Status = "failed"
	}
	switch moduleID {
	case "m1_arch":
		return strings.TrimSpace(fmt.Sprintf(`
请作为 M1 裁判严格评分。你必须遵循以下规则文件：
- @modules/m1_arch/JUDGE_AGENT.md

评分输入材料：
- @eval_records/%[1]s/m1_arch/m1_result.proto
- @eval_records/%[1]s/m1_arch/m1_build.log
- @eval_records/%[1]s/m1_arch/m1_test.log

执行状态（必须作为评分依据）：
- harness_status = %[7]s
- harness_error = %[8]q

运行时长指标（必须在输出 JSON 中填写 runtime_metrics）：
- phase1_seconds = %[3].1f
- phase2_seconds = %[4].1f
- phase3_seconds = %[5].1f
- total_seconds = %[6].1f

输出要求（强制）：
1) 只输出 JSON，禁止 markdown 和额外说明
2) 按 100 分制给出总分与分项
3) 重点评估：协议抽象能力、字段演进正确性、接口收敛质量
4) 规约测试失败时必须“扣分但仍给结果”，不得因为 FAIL 直接拒绝评分
5) 如果怀疑 harness 存在语义误伤，允许结合代码语义做纠偏，但要在 final_reasoning 说明依据
6) 轻微工程错误（如 unused import）只应影响相关维度，禁止将总分直接清零
4) 在 JSON 中填写：
   - "judge_model": "%[2]s"
   - "runtime_metrics": {"phase1_seconds": x, "phase2_seconds": y, "phase3_seconds": z, "total_seconds": t}
`, modelDir, judgeModel, runtime.Phase1Seconds, runtime.Phase2Seconds, runtime.Phase3Seconds, runtime.TotalSeconds, phase2Status, phase2Error)), nil
	case "m2_biz":
		return strings.TrimSpace(fmt.Sprintf(`
请作为 M2 裁判严格评分。你必须遵循以下规则文件：
- @modules/m2_biz/JUDGE_AGENT.md

评分输入材料：
- @eval_records/%[1]s/m2_biz/m2_result.go
- @eval_records/%[1]s/m2_biz/m2_build.log
- @eval_records/%[1]s/m2_biz/m2_test.log

执行状态（必须作为评分依据）：
- harness_status = %[7]s
- harness_error = %[8]q

运行时长指标（必须在输出 JSON 中填写 runtime_metrics）：
- phase1_seconds = %[3].1f
- phase2_seconds = %[4].1f
- phase3_seconds = %[5].1f
- total_seconds = %[6].1f

输出要求（强制）：
1) 只输出 JSON，禁止 markdown 和额外说明
2) 按 100 分制给出总分与分项
3) 必须检查事务一致性、Kafka 失败补偿与 ctx 透传
4) 规约测试失败时必须“扣分但仍给结果”，不得因为 FAIL 直接拒绝评分
5) 如果怀疑 harness 存在语义误伤，允许结合代码语义做纠偏，但要在 final_reasoning 说明依据
6) 轻微工程错误（如 unused import）只应影响相关维度，禁止将总分直接清零
7) 在 JSON 中填写：
   - "judge_model": "%[2]s"
   - "runtime_metrics": {"phase1_seconds": x, "phase2_seconds": y, "phase3_seconds": z, "total_seconds": t}
`, modelDir, judgeModel, runtime.Phase1Seconds, runtime.Phase2Seconds, runtime.Phase3Seconds, runtime.TotalSeconds, phase2Status, phase2Error)), nil
	case "m3_component":
		return strings.TrimSpace(fmt.Sprintf(`
请作为 M3 裁判严格评分。你必须遵循以下规则文件：
- @templates/phase2_judge_prompt.md
- @modules/m3_component/JUDGE_AGENT.md

评分输入材料：
- @eval_records/%[1]s/m3_component/m3_result.go
- @eval_records/%[1]s/m3_component/m3_build.log
- @eval_records/%[1]s/m3_component/m3_test.log

执行状态（必须作为评分依据）：
- harness_status = %[7]s
- harness_error = %[8]q

运行时长指标（必须在输出 JSON 中填写 runtime_metrics）：
- phase1_seconds = %[3].1f
- phase2_seconds = %[4].1f
- phase3_seconds = %[5].1f
- total_seconds = %[6].1f

输出要求（强制）：
1) 只输出 JSON，禁止 markdown 和额外说明
2) 按 100 分制给出总分与分项
3) 必须检查是否存在 // BUGFIX: 注释，否则 D3 至少扣 8 分
4) 规约测试失败时必须“扣分但仍给结果”，不得因为 FAIL 直接拒绝评分
5) 如果怀疑 harness 存在语义误伤，允许结合代码语义做纠偏，但要在 final_reasoning 说明依据
6) 轻微工程错误（如 unused import）只应影响相关维度，禁止将总分直接清零
7) 在 JSON 中填写：
   - "judge_model": "%[2]s"
   - "runtime_metrics": {"phase1_seconds": x, "phase2_seconds": y, "phase3_seconds": z, "total_seconds": t}
`, modelDir, judgeModel, runtime.Phase1Seconds, runtime.Phase2Seconds, runtime.Phase3Seconds, runtime.TotalSeconds, phase2Status, phase2Error)), nil
	case "m4_bugfix":
		return strings.TrimSpace(fmt.Sprintf(`
请作为 M4 裁判严格评分。你必须遵循以下规则文件：
- @templates/phase2_judge_prompt.md
- @modules/m4_bugfix/JUDGE_AGENT.md

评分输入材料：
- @eval_records/%[1]s/m4_bugfix/m4_result.go
- @eval_records/%[1]s/m4_bugfix/m4_build.log
- @eval_records/%[1]s/m4_bugfix/m4_test.log

执行状态（必须作为评分依据）：
- harness_status = %[7]s
- harness_error = %[8]q

运行时长指标（必须在输出 JSON 中填写 runtime_metrics）：
- phase1_seconds = %[3].1f
- phase2_seconds = %[4].1f
- phase3_seconds = %[5].1f
- total_seconds = %[6].1f

输出要求（强制）：
1) 只输出 JSON，禁止 markdown 和额外说明
2) 按 100 分制给出总分与分项
3) 必须检查是否存在 // BUGFIX: 注释，否则 D3 记 0
4) 规约测试失败时必须“扣分但仍给结果”，不得因为 FAIL 直接拒绝评分
5) 如果怀疑 harness 存在语义误伤，允许结合代码语义做纠偏，但要在 final_reasoning 说明依据
6) 轻微工程错误（如 unused import）只应影响相关维度，禁止将总分直接清零
7) 在 JSON 中填写：
   - "judge_model": "%[2]s"
   - "runtime_metrics": {"phase1_seconds": x, "phase2_seconds": y, "phase3_seconds": z, "total_seconds": t}
`, modelDir, judgeModel, runtime.Phase1Seconds, runtime.Phase2Seconds, runtime.Phase3Seconds, runtime.TotalSeconds, phase2Status, phase2Error)), nil
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

func extractTaggedCodeBlocks(output string) map[string]string {
	re := regexp.MustCompile("(?s)```([a-zA-Z0-9_+-]+)\\n(.*?)```")
	matches := re.FindAllStringSubmatch(output, -1)
	blocks := make(map[string]string, len(matches))
	for _, m := range matches {
		if len(m) < 3 {
			continue
		}
		tag := strings.TrimSpace(m[1])
		if tag == "" {
			continue
		}
		blocks[tag] = strings.TrimSpace(m[2]) + "\n"
	}
	return blocks
}

func phase1PromptForModules(modules []string) (string, error) {
	supported := map[string]struct{}{
		"m1_arch":      {},
		"m2_biz":       {},
		"m3_component": {},
		"m4_bugfix":    {},
	}
	for _, m := range modules {
		if _, ok := supported[m]; !ok {
			return "", fmt.Errorf("unsupported module for batch phase1 prompt: %s", m)
		}
	}
	var b strings.Builder
	b.WriteString("你现在在多模块综合评测中。必须一次性产出所有指定模块结果，禁止解释文字。\n\n")
	b.WriteString("可见文件（仅限 input 与规则）：\n")
	for _, m := range modules {
		switch m {
		case "m1_arch":
			b.WriteString("- @modules/m1_arch/input/prd.md\n")
			b.WriteString("- @modules/m1_arch/input/api.proto\n")
			b.WriteString("- @modules/m1_arch/input/guidance.md\n")
			b.WriteString("- @modules/m1_arch/.cursorrules\n")
		case "m2_biz":
			b.WriteString("- @modules/m2_biz/input/api.proto\n")
			b.WriteString("- @modules/m2_biz/input/biz_spec.md\n")
			b.WriteString("- @modules/m2_biz/input/interfaces.go\n")
			b.WriteString("- @modules/m2_biz/input/guidance.md\n")
			b.WriteString("- @modules/m2_biz/.cursorrules\n")
		case "m3_component":
			b.WriteString("- @modules/m3_component/input/guidance.md\n")
			b.WriteString("- @modules/m3_component/input/rate_limit_spec.md\n")
			b.WriteString("- @modules/m3_component/.cursorrules\n")
		case "m4_bugfix":
			b.WriteString("- @modules/m4_bugfix/input/guidance.md\n")
			b.WriteString("- @modules/m4_bugfix/input/bug_report.md\n")
			b.WriteString("- @modules/m4_bugfix/input/legacy_code.go\n")
			b.WriteString("- @modules/m4_bugfix/.cursorrules\n")
		}
	}
	b.WriteString("\n输出要求（强制）：\n")
	b.WriteString("1) 只输出带标签的代码块，不要输出任何解释。\n")
	b.WriteString("2) 每个模块只输出一个结果代码块，标签与模块一一对应：\n")
	if slices.Contains(modules, "m1_arch") {
		b.WriteString("   - ```m1_proto 对应 m1_arch 的完整 proto\n")
	}
	if slices.Contains(modules, "m2_biz") {
		b.WriteString("   - ```m2_go 对应 m2_biz 的业务实现（package result）\n")
	}
	if slices.Contains(modules, "m3_component") {
		b.WriteString("   - ```m3_go 对应 m3_component 的组件实现（package result）\n")
	}
	if slices.Contains(modules, "m4_bugfix") {
		b.WriteString("   - ```m4_go 对应 m4_bugfix 的修复后实现（package result）\n")
	}
	b.WriteString("3) 严禁漏掉任何指定模块。\n")
	return strings.TrimSpace(b.String()), nil
}

func phase3PromptForModules(
	modules []string,
	modelDir, judgeModel string,
	phase2Status map[string]string,
	phase2Error map[string]string,
	runtime map[string]module.RuntimeMetrics,
) (string, error) {
	var b strings.Builder
	b.WriteString("请作为统一裁判，一次性评估多个模块，输出一个 JSON 对象。\n")
	b.WriteString("必须遵循对应模块的 JUDGE_AGENT 规则文件。\n\n")
	for _, m := range modules {
		switch m {
		case "m1_arch":
			b.WriteString("- @modules/m1_arch/JUDGE_AGENT.md\n")
		case "m2_biz":
			b.WriteString("- @modules/m2_biz/JUDGE_AGENT.md\n")
		case "m3_component":
			b.WriteString("- @modules/m3_component/JUDGE_AGENT.md\n")
		case "m4_bugfix":
			b.WriteString("- @modules/m4_bugfix/JUDGE_AGENT.md\n")
		default:
			return "", fmt.Errorf("unsupported module for batch phase3 prompt: %s", m)
		}
	}
	b.WriteString("\n评分输入材料：\n")
	for _, m := range modules {
		switch m {
		case "m1_arch":
			b.WriteString(fmt.Sprintf("- @eval_records/%s/m1_arch/m1_result.proto\n", modelDir))
			b.WriteString(fmt.Sprintf("- @eval_records/%s/m1_arch/m1_build.log\n", modelDir))
			b.WriteString(fmt.Sprintf("- @eval_records/%s/m1_arch/m1_test.log\n", modelDir))
		case "m2_biz":
			b.WriteString(fmt.Sprintf("- @eval_records/%s/m2_biz/m2_result.go\n", modelDir))
			b.WriteString(fmt.Sprintf("- @eval_records/%s/m2_biz/m2_build.log\n", modelDir))
			b.WriteString(fmt.Sprintf("- @eval_records/%s/m2_biz/m2_test.log\n", modelDir))
		case "m3_component":
			b.WriteString(fmt.Sprintf("- @eval_records/%s/m3_component/m3_result.go\n", modelDir))
			b.WriteString(fmt.Sprintf("- @eval_records/%s/m3_component/m3_build.log\n", modelDir))
			b.WriteString(fmt.Sprintf("- @eval_records/%s/m3_component/m3_test.log\n", modelDir))
		case "m4_bugfix":
			b.WriteString(fmt.Sprintf("- @eval_records/%s/m4_bugfix/m4_result.go\n", modelDir))
			b.WriteString(fmt.Sprintf("- @eval_records/%s/m4_bugfix/m4_build.log\n", modelDir))
			b.WriteString(fmt.Sprintf("- @eval_records/%s/m4_bugfix/m4_test.log\n", modelDir))
		}
	}
	b.WriteString("\n执行状态（必须作为评分依据）：\n")
	for _, m := range modules {
		rt := runtime[m]
		b.WriteString(fmt.Sprintf("- %s: harness_status=%s, harness_error=%q, phase1=%.1f, phase2=%.1f\n",
			m, phase2Status[m], phase2Error[m], rt.Phase1Seconds, rt.Phase2Seconds))
	}
	b.WriteString("\n评分原则（强制）：\n")
	b.WriteString("1) harness 失败时必须扣分但仍给分，不得拒绝输出。\n")
	b.WriteString("2) 若怀疑存在语义误伤，可结合代码语义纠偏，并在理由中解释。\n")
	b.WriteString("3) 必须按各模块 JUDGE_AGENT 的维度和权重打分，不得强行统一成 25/25/25/25。\n")
	b.WriteString("4) 轻微工程错误（如 unused import）仅影响对应维度，禁止因此将模块总分直接清零。\n")
	b.WriteString("5) 必须只输出一个合法 JSON 对象，禁止 markdown、禁止代码块、禁止额外解释文字。\n")
	b.WriteString("6) 每个 breakdown 项都要给出：dimension/score/max_score/log_evidence/code_evidence。\n")
	b.WriteString("7) 输出 JSON 必须使用以下结构（字段名保持一致，模块分项维度可自定义）：\n")
	b.WriteString(fmt.Sprintf("{\"model\":\"%s\",\"judge_model\":\"%s\",\"module_scores\":{\"m1_arch\":{\"total_score\":0,\"breakdown\":{\"<custom_key>\":{\"dimension\":\"\",\"score\":0,\"max_score\":0,\"log_evidence\":\"\",\"code_evidence\":\"\"}},\"final_reasoning\":\"\"},\"m2_biz\":{\"total_score\":0,\"breakdown\":{\"<custom_key>\":{\"dimension\":\"\",\"score\":0,\"max_score\":0,\"log_evidence\":\"\",\"code_evidence\":\"\"}},\"final_reasoning\":\"\"},\"m3_component\":{\"total_score\":0,\"breakdown\":{\"<custom_key>\":{\"dimension\":\"\",\"score\":0,\"max_score\":0,\"log_evidence\":\"\",\"code_evidence\":\"\"}},\"final_reasoning\":\"\"},\"m4_bugfix\":{\"total_score\":0,\"breakdown\":{\"<custom_key>\":{\"dimension\":\"\",\"score\":0,\"max_score\":0,\"log_evidence\":\"\",\"code_evidence\":\"\"}},\"final_reasoning\":\"\"}},\"final_reasoning\":\"\"}", modelDir, judgeModel))
	return b.String(), nil
}
