package cmd

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
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
	b.WriteString("- 记录模型在每个模块生成产物的 Token 开销\n")
	b.WriteString("- 单位：*K* Token\n\n")
	b.WriteString("| 模型 | M1 | M2 | M3 | M4 |\n")
	b.WriteString("| --- | --- | --- | --- | --- |\n")
	for _, m := range models {
		b.WriteString("| " + escapePipe(m.name) + " | - | - | - | - |\n")
	}
	if len(models) == 0 {
		b.WriteString("| - | - | - | - | - |\n")
	}

	return b.String()
}

func renderCorrectnessSubTable(b *strings.Builder, models []modelScores, moduleLabel, moduleID string) {
	b.WriteString("### " + moduleLabel + "\n\n")
	b.WriteString("| 模型 | 总分 | 编译 | 测试 | 静态 | 运行时 |\n")
	b.WriteString("| --- | --- | --- | --- | --- | --- |\n")
	for _, m := range models {
		s := scoreByModule(m, moduleID)
		b.WriteString(fmt.Sprintf(
			"| %s | %d | %d | %d | %d | %d |\n",
			escapePipe(m.name),
			moduleTotalByRule(s),
			breakdownScoreOrZero(s, "execution_compile"),
			breakdownScoreOrZero(s, "execution_test"),
			breakdownScoreOrZero(s, "static_analysis"),
			breakdownScoreOrZero(s, "execution_runtime"),
		))
	}
	if len(models) == 0 {
		b.WriteString("| - | 0 | 0 | 0 | 0 | 0 |\n")
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
	compile := breakdownScoreOrZero(s, "execution_compile")
	if compile == 0 {
		return 0
	}
	return s.TotalScore
}

func escapePipe(v string) string {
	return strings.ReplaceAll(v, "|", "\\|")
}
