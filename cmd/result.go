package cmd

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"ai_eval/internal/module"
)

var moduleOrder = []string{
	"m1_arch",
	"m2_biz",
	"m3_component",
	"m4_bugfix",
}

type modelScores struct {
	modelDir string
	name     string
	byModule map[string]module.Score
}

func runResult(args []string) int {
	fs := flag.NewFlagSet("result", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		recordsDir = fs.String("dir", "eval_records", "eval records root directory")
		outPath    = fs.String("out", "RESULT.md", "output markdown file path")
	)
	if err := fs.Parse(args); err != nil {
		return 2
	}

	models, err := collectScores(*recordsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "collect score json failed: %v\n", err)
		return 1
	}

	content := renderResultMarkdown(models)
	if err := os.WriteFile(*outPath, []byte(content), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write result markdown failed: %v\n", err)
		return 1
	}

	fmt.Println("ai_eval result success")
	fmt.Printf("records dir: %s\n", *recordsDir)
	fmt.Printf("models: %d\n", len(models))
	fmt.Printf("output: %s\n", *outPath)
	return 0
}

func collectScores(recordsDir string) ([]modelScores, error) {
	agg := map[string]*modelScores{}
	err := filepath.WalkDir(recordsDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || d.Name() != "score.json" {
			return nil
		}

		rel, err := filepath.Rel(recordsDir, path)
		if err != nil {
			return err
		}
		parts := strings.Split(filepath.ToSlash(rel), "/")
		if len(parts) != 3 {
			return nil
		}
		modelDir := strings.TrimSpace(parts[0])
		moduleID := strings.TrimSpace(parts[1])
		if modelDir == "" || moduleID == "" {
			return nil
		}

		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var s module.Score
		if err := json.Unmarshal(raw, &s); err != nil {
			return err
		}
		s = normalizeLegacyScoreShape(s, raw, moduleID)

		ms, ok := agg[modelDir]
		if !ok {
			ms = &modelScores{
				modelDir: modelDir,
				name:     modelDir,
				byModule: map[string]module.Score{},
			}
			agg[modelDir] = ms
		}
		if strings.TrimSpace(s.Model) != "" {
			ms.name = s.Model
		}
		ms.byModule[moduleID] = s
		return nil
	})
	if err != nil {
		return nil, err
	}

	out := make([]modelScores, 0, len(agg))
	for _, v := range agg {
		out = append(out, *v)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].name < out[j].name
	})
	return out, nil
}

func normalizeLegacyScoreShape(s module.Score, raw []byte, moduleID string) module.Score {
	if len(s.Breakdown) > 0 {
		return s
	}
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return s
	}
	breakdown := map[string]module.ScoreDetail{}
	for i := 1; i <= 9; i++ {
		scoreKey := fmt.Sprintf("D%d_score", i)
		reasonKey := fmt.Sprintf("D%d_reason", i)
		rawScore, ok := obj[scoreKey]
		if !ok {
			continue
		}
		scoreVal := int(anyNumber(rawScore))
		dKey := fmt.Sprintf("D%d", i)
		detail := module.ScoreDetail{
			Dimension:   legacyDimensionByModule(moduleID, dKey),
			Score:       scoreVal,
			MaxScore:    legacyMaxByModule(moduleID, dKey),
			LogEvidence: anyString(obj[reasonKey]),
		}
		if detail.Dimension == "" {
			detail.Dimension = dKey
		}
		breakdown[dKey] = detail
	}
	if len(breakdown) > 0 {
		s.Breakdown = breakdown
	}
	return s
}

func legacyMaxByModule(moduleID, dKey string) int {
	table := map[string]map[string]int{
		"m1_arch": {
			"D1": 15, "D2": 35, "D3": 25, "D4": 25,
		},
		"m2_biz": {
			"D1": 15, "D2": 30, "D3": 25, "D4": 15, "D5": 15,
		},
		"m3_component": {
			"D1": 15, "D2": 30, "D3": 25, "D4": 15, "D5": 15,
		},
		"m4_bugfix": {
			"D1": 10, "D2": 40, "D3": 25, "D4": 15, "D5": 10,
		},
	}
	if t, ok := table[moduleID]; ok {
		return t[dKey]
	}
	return 0
}

func legacyDimensionByModule(moduleID, dKey string) string {
	table := map[string]map[string]string{
		"m1_arch": {
			"D1": "协议基础合法性与编译",
			"D2": "业务功能结构抽象",
			"D3": "接口聚合与收敛设计",
			"D4": "历史包袱与防御性设计",
		},
		"m2_biz": {
			"D1": "编译通过率",
			"D2": "业务功能与并发防刷防御",
			"D3": "事务保护与状态一致性",
			"D4": "分布式降级与重试容错",
			"D5": "可验证性与工程质量",
		},
		"m3_component": {
			"D1": "编译通过率",
			"D2": "惊群防御与 Singleflight 级抽象",
			"D3": "细粒度锁与慢阻塞隔离",
			"D4": "防御性编程与资源控制",
			"D5": "可验证性与性能意识",
		},
		"m4_bugfix": {
			"D1": "编译通过率",
			"D2": "隐患挖掘与修复质量",
			"D3": "规约验证结果",
			"D4": "历史包袱约束遵守",
			"D5": "代码性能与工程整洁度",
		},
	}
	if t, ok := table[moduleID]; ok {
		return t[dKey]
	}
	return dKey
}

func anyNumber(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case json.Number:
		f, _ := n.Float64()
		return f
	default:
		return 0
	}
}

func anyString(v any) string {
	s, _ := v.(string)
	return strings.TrimSpace(s)
}

func renderResultMarkdown(models []modelScores) string {
	var b strings.Builder
	b.WriteString("# RESULT\n\n")
	b.WriteString("## 1. 正确性\n")
	b.WriteString("- 评测标准：每个模块评判的总分，缺失或者编译失败计 0 分\n")
	b.WriteString("- 权重：25%；25%；25%；25%；\n\n")

	renderCorrectnessSubTable(&b, models, "M1", "m1_arch")
	renderCorrectnessSubTable(&b, models, "M2", "m2_biz")
	renderCorrectnessSubTable(&b, models, "M3", "m3_component")
	renderCorrectnessSubTable(&b, models, "M4", "m4_bugfix")

	b.WriteString("### 总分汇总\n\n")
	b.WriteString("| 模型 | M1总分 | M2总分 | M3总分 | M4总分 | 加权总分 |\n")
	b.WriteString("| --- | --- | --- | --- | --- | --- |\n")
	for _, m := range models {
		m1 := moduleTotalByRule(scoreByModule(m, "m1_arch"))
		m2 := moduleTotalByRule(scoreByModule(m, "m2_biz"))
		m3 := moduleTotalByRule(scoreByModule(m, "m3_component"))
		m4 := moduleTotalByRule(scoreByModule(m, "m4_bugfix"))
		b.WriteString(fmt.Sprintf(
			"| %s | %d | %d | %d | %d | %.1f |\n",
			escapePipe(m.name),
			m1, m2, m3, m4,
			weightedTotal(m),
		))
	}
	if len(models) == 0 {
		b.WriteString("| - | 0 | 0 | 0 | 0 | 0.0 |\n")
	}

	b.WriteString("\n## 2. 时间\n")
	b.WriteString("- 记录模型在每个模块生成产物的时间开销\n")
	b.WriteString("- 单位：秒\n\n")
	b.WriteString("| 模型 | M1 | M2 | M3 | M4 |\n")
	b.WriteString("| --- | --- | --- | --- | --- |\n")
	for _, m := range models {
		row := []string{escapePipe(m.name)}
		for _, moduleID := range moduleOrder {
			s, ok := m.byModule[moduleID]
			if !ok {
				row = append(row, "-")
				continue
			}
			row = append(row, fmt.Sprintf("%.1f", s.RuntimeMetrics.Phase1Seconds))
		}
		b.WriteString("| " + strings.Join(row, " | ") + " |\n")
	}
	if len(models) == 0 {
		b.WriteString("| - | - | - | - | - |\n")
	}

	b.WriteString("\n## 3. 成本\n")
	b.WriteString("- 记录模型整轮评测的总 Token 开销\n")
	b.WriteString("- 单位：*K* Token\n\n")
	b.WriteString("| 模型 | Total |\n")
	b.WriteString("| --- | --- |\n")
	for _, m := range models {
		b.WriteString("| " + escapePipe(m.name) + " | - |\n")
	}
	if len(models) == 0 {
		b.WriteString("| - | - |\n")
	}

	return b.String()
}

func renderCorrectnessSubTable(b *strings.Builder, models []modelScores, moduleLabel, moduleID string) {
	columns, meta := ruleColumnsAndMeta(models, moduleID)
	explicitD := hasExplicitDRules(models, moduleID)
	b.WriteString("### " + moduleLabel + "\n\n")
	header := []string{"模型"}
	for _, col := range columns {
		header = append(header, col)
	}
	header = append(header, "总分")
	b.WriteString("| " + strings.Join(header, " | ") + " |\n")
	sep := make([]string, 0, len(header))
	for range header {
		sep = append(sep, "---")
	}
	b.WriteString("| " + strings.Join(sep, " | ") + " |\n")
	for _, m := range models {
		s := scoreByModule(m, moduleID)
		row := []string{escapePipe(m.name)}
		for _, col := range columns {
			row = append(row, fmt.Sprintf("%d", scoreByColumn(s, col, explicitD)))
		}
		row = append(row, fmt.Sprintf("%d", moduleTotalByRule(s)))
		b.WriteString("| " + strings.Join(row, " | ") + " |\n")
	}
	if len(models) == 0 {
		row := []string{"-"}
		for range columns {
			row = append(row, "0")
		}
		row = append(row, "0")
		b.WriteString("| " + strings.Join(row, " | ") + " |\n")
	}
	if len(columns) > 0 {
		labels := make([]string, 0, len(meta))
		for _, col := range columns {
			m := meta[col]
			if m.maxScore > 0 {
				labels = append(labels, fmt.Sprintf("- %s：%s（满分 %d 分）", col, m.dimension, m.maxScore))
			} else {
				labels = append(labels, fmt.Sprintf("- %s：%s", col, m.dimension))
			}
		}
		b.WriteString("\n规则说明：\n" + strings.Join(labels, "\n") + "\n")
	}
	b.WriteString("\n")
}

func weightedTotal(m modelScores) float64 {
	var total float64
	for _, moduleID := range moduleOrder {
		total += float64(moduleTotalByRule(scoreByModule(m, moduleID))) * 0.25
	}
	return total
}

func breakdownScoreOrZero(s module.Score, key string) int {
	d, ok := s.Breakdown[key]
	if !ok {
		return 0
	}
	return d.Score
}

func scoreByModule(m modelScores, moduleID string) module.Score {
	s, ok := m.byModule[moduleID]
	if !ok {
		return module.Score{}
	}
	return s
}

func moduleTotalByRule(s module.Score) int {
	return s.TotalScore
}

type ruleMeta struct {
	dimension string
	maxScore  int
}

func scoreKeyRank(k string) (rank int, num int) {
	if strings.HasPrefix(k, "D") && len(k) > 1 {
		n, err := strconv.Atoi(strings.TrimPrefix(k, "D"))
		if err == nil {
			return 0, n
		}
	}
	switch k {
	case "execution_compile":
		return 1, 0
	case "execution_test":
		return 1, 1
	case "static_analysis":
		return 1, 2
	case "execution_runtime":
		return 1, 3
	default:
		return 2, 0
	}
}

func ruleColumnsAndMeta(models []modelScores, moduleID string) ([]string, map[string]ruleMeta) {
	explicitD := hasExplicitDRules(models, moduleID)
	meta := map[string]ruleMeta{}
	seen := map[string]struct{}{}
	out := make([]string, 0)
	for _, m := range models {
		s := scoreByModule(m, moduleID)
		for k, d := range s.Breakdown {
			label := normalizeRuleLabel(k, d.Dimension, explicitD)
			if label == "" {
				continue
			}
			if _, ok := seen[label]; !ok {
				seen[label] = struct{}{}
				out = append(out, label)
			}
			existing := meta[label]
			if strings.TrimSpace(existing.dimension) == "" && strings.TrimSpace(d.Dimension) != "" {
				existing.dimension = strings.TrimSpace(d.Dimension)
			}
			if d.MaxScore > existing.maxScore {
				existing.maxScore = d.MaxScore
			}
			meta[label] = existing
		}
	}
	sort.Slice(out, func(i, j int) bool {
		ri, ni := scoreKeyRank(out[i])
		rj, nj := scoreKeyRank(out[j])
		if ri != rj {
			return ri < rj
		}
		if ni != nj {
			return ni < nj
		}
		return out[i] < out[j]
	})
	return out, meta
}

func scoreByColumn(s module.Score, col string, explicitD bool) int {
	for k, d := range s.Breakdown {
		if normalizeRuleLabel(k, d.Dimension, explicitD) == col {
			return d.Score
		}
	}
	return 0
}

func normalizeDKey(key string) (string, bool) {
	if strings.HasPrefix(key, "D") && len(key) > 1 {
		n := ""
		for _, ch := range strings.TrimPrefix(key, "D") {
			if ch >= '0' && ch <= '9' {
				n += string(ch)
				continue
			}
			break
		}
		if n != "" {
			return "D" + n, true
		}
	}
	return "", false
}

func normalizeRuleLabel(key, dimension string, explicitD bool) string {
	if dk, ok := normalizeDKey(strings.TrimSpace(key)); ok {
		return dk
	}
	if explicitD {
		return ""
	}
	name := strings.ToLower(strings.TrimSpace(dimension))
	switch {
	case strings.Contains(name, "编译"), strings.Contains(name, "compile"):
		return "D1"
	case strings.Contains(name, "测试"), strings.Contains(name, "test"), strings.Contains(name, "功能"):
		return "D2"
	case strings.Contains(name, "静态"), strings.Contains(name, "规范"), strings.Contains(name, "analysis"), strings.Contains(name, "代码"):
		return "D3"
	case strings.Contains(name, "运行时"), strings.Contains(name, "性能"), strings.Contains(name, "runtime"), strings.Contains(name, "efficiency"):
		return "D4"
	default:
		return ""
	}
}

func hasExplicitDRules(models []modelScores, moduleID string) bool {
	for _, m := range models {
		s := scoreByModule(m, moduleID)
		for k := range s.Breakdown {
			if _, ok := normalizeDKey(k); ok {
				return true
			}
		}
	}
	return false
}

func escapePipe(v string) string {
	return strings.ReplaceAll(v, "|", "\\|")
}
