package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ai_eval/internal/eval"
)

func main() {
	var (
		modelsFlag  = flag.String("models", "", "comma-separated candidate models")
		modulesFlag = flag.String("modules", "m4", "comma-separated modules, e.g. m4 or m1,m2,m3,m4")
	)
	flag.Parse()

	models := splitCSV(*modelsFlag)
	if len(models) == 0 {
		fmt.Fprintln(os.Stderr, "usage: ai_eval_init --models \"model-a,model-b\" [--modules \"m4\"]")
		os.Exit(2)
	}
	modules, err := normalizeModules(splitCSV(*modulesFlag))
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid modules: %v\n", err)
		os.Exit(2)
	}

	for _, model := range models {
		modelDir := eval.ModelDirName(model)
		for _, module := range modules {
			recordDir := filepath.Join("eval_records", modelDir, module)
			if err := os.MkdirAll(recordDir, 0o755); err != nil {
				fmt.Fprintf(os.Stderr, "mkdir failed: %s: %v\n", recordDir, err)
				os.Exit(1)
			}
		}
	}

	fmt.Println("ai_eval_init success")
	fmt.Printf("models: %s\n", strings.Join(models, ", "))
	fmt.Printf("modules: %s\n", strings.Join(modules, ", "))
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}

func normalizeModules(modules []string) ([]string, error) {
	if len(modules) == 0 {
		return nil, fmt.Errorf("empty modules")
	}
	out := make([]string, 0, len(modules))
	seen := map[string]struct{}{}
	for _, module := range modules {
		mapped, err := normalizeModule(module)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[mapped]; ok {
			continue
		}
		seen[mapped] = struct{}{}
		out = append(out, mapped)
	}
	return out, nil
}

func normalizeModule(module string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(module)) {
	case "m1", "m1_arch":
		return "m1_arch", nil
	case "m2", "m2_biz":
		return "m2_biz", nil
	case "m3", "m3_component":
		return "m3_component", nil
	case "m4", "m4_bugfix":
		return "m4_bugfix", nil
	default:
		return "", fmt.Errorf("unsupported module: %s", module)
	}
}
