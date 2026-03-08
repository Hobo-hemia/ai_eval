package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ai_eval/internal/eval"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "init":
			os.Exit(runInit(os.Args[2:]))
		case "run":
			os.Exit(runRun(os.Args[2:]))
		case "clear":
			os.Exit(runClear(os.Args[2:]))
		case "-h", "--help", "help":
			printRootUsage()
			return
		}
	}

	// Backward-compatible mode: ai_eval --module ... --model ...
	os.Exit(runRun(os.Args[1:]))
}

func runRun(args []string) int {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		module     = fs.String("module", "", "module id, currently supports: m4|m4_bugfix")
		model      = fs.String("model", "", "model id to evaluate")
		judgeModel = fs.String("judge-model", "gemini-3-flash", "judge model id")
		cursorBin  = fs.String("cursor-bin", "cursor-agent", "cursor CLI binary path")
	)
	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *module == "" || *model == "" {
		fmt.Fprintln(os.Stderr, "usage: ai_eval run --module m4 --model <MODEL_X> [--judge-model <JUDGE_MODEL>]")
		return 2
	}

	result, err := eval.RunAutoEvaluation(context.Background(), eval.AutoRunConfig{
		WorkspaceRoot: ".",
		CursorBin:     *cursorBin,
		Model:         *model,
		Module:        *module,
		JudgeModel:    *judgeModel,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "ai_eval run failed: %v\n", err)
		return 1
	}

	fmt.Println("ai_eval run success")
	fmt.Printf("model: %s\n", result.Model)
	fmt.Printf("model dir: %s\n", result.ModelDir)
	fmt.Printf("module: %s\n", result.Module)
	fmt.Printf("judge model: %s\n", result.JudgeModel)
	fmt.Printf("result: %s\n", result.ResultFile)
	fmt.Printf("build log: %s\n", result.BuildLogFile)
	fmt.Printf("test log: %s\n", result.TestLogFile)
	fmt.Printf("score: %s\n", result.ScoreFile)
	return 0
}

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

	for _, model := range models {
		modelDir := eval.ModelDirName(model)
		for _, module := range modules {
			recordDir := filepath.Join("eval_records", modelDir, module)
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

func runClear(args []string) int {
	fs := flag.NewFlagSet("clear", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		evalRecordsDir = fs.String("dir", "eval_records", "eval records root directory")
		keepReadme     = fs.Bool("keep-readme", true, "keep eval_records/README.md")
	)
	if err := fs.Parse(args); err != nil {
		return 2
	}

	entries, err := os.ReadDir(*evalRecordsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read eval_records failed: %v\n", err)
		return 1
	}

	removed := 0
	for _, entry := range entries {
		name := entry.Name()
		if *keepReadme && name == "README.md" {
			continue
		}
		target := filepath.Join(*evalRecordsDir, name)
		if err := os.RemoveAll(target); err != nil {
			fmt.Fprintf(os.Stderr, "remove failed: %s: %v\n", target, err)
			return 1
		}
		removed++
	}
	fmt.Println("ai_eval clear success")
	fmt.Printf("removed entries: %d\n", removed)
	fmt.Printf("target dir: %s\n", *evalRecordsDir)
	return 0
}

func printRootUsage() {
	fmt.Println("ai_eval: unified evaluator CLI")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  ai_eval init --models \"m1,m2\" [--modules \"m4\"]")
	fmt.Println("  ai_eval run --module m4 --model \"gpt-5.3-codex\" [--judge-model \"gemini-3-flash\"]")
	fmt.Println("  ai_eval clear [--dir eval_records] [--keep-readme=true]")
	fmt.Println("")
	fmt.Println("Backward-compatible:")
	fmt.Println("  ai_eval --module m4 --model \"gpt-5.3-codex\"")
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
