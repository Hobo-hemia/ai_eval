package cmd

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	core "ai_eval/internal"
)

func runInit(args []string) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		modelsFlag  = fs.String("models", "", "comma-separated candidate models")
		modulesFlag = fs.String("modules", "m4", "comma-separated modules, e.g. m4 or m1,m2,m3,m4")
	)
	if err := fs.Parse(args); err != nil {
		return 2
	}

	models := splitCSV(*modelsFlag)
	if len(models) == 0 {
		fmt.Fprintln(os.Stderr, "usage: ai_eval init --models \"model-a,model-b\" [--modules \"m4\"]")
		return 2
	}
	modules, err := normalizeModules(splitCSV(*modulesFlag))
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid modules: %v\n", err)
		return 2
	}

	for _, modelName := range models {
		modelDir := core.ModelDirName(modelName)
		for _, moduleID := range modules {
			recordDir := filepath.Join("eval_records", modelDir, moduleID)
			if err := os.MkdirAll(recordDir, 0o755); err != nil {
				fmt.Fprintf(os.Stderr, "mkdir failed: %s: %v\n", recordDir, err)
				return 1
			}
		}
	}

	fmt.Println("ai_eval init success")
	fmt.Printf("models: %s\n", strings.Join(models, ", "))
	fmt.Printf("modules: %s\n", strings.Join(modules, ", "))
	return 0
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
	for _, moduleID := range modules {
		mapped, err := normalizeModule(moduleID)
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

func normalizeModule(moduleID string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(moduleID)) {
	case "m1", "m1_arch":
		return "m1_arch", nil
	case "m2", "m2_biz":
		return "m2_biz", nil
	case "m3", "m3_component":
		return "m3_component", nil
	case "m4", "m4_bugfix":
		return "m4_bugfix", nil
	default:
		return "", fmt.Errorf("unsupported module: %s", moduleID)
	}
}
