package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"

	core "ai_eval/internal"
)

func Execute(args []string) int {
	if len(args) > 0 {
		switch args[0] {
		case "init":
			return runInit(args[1:])
		case "run":
			return runRun(args[1:])
		case "clear":
			return runClear(args[1:])
		case "-h", "--help", "help":
			printUsage()
			return 0
		}
	}
	// Backward-compatible mode: ai_eval --module ... --model ...
	return runRun(args)
}

func runRun(args []string) int {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		moduleID   = fs.String("module", "", "module id, currently supports: m3|m3_component|m4|m4_bugfix")
		modelName  = fs.String("model", "", "model id to evaluate")
		judgeModel = fs.String("judge-model", "gemini-3-flash", "judge model id")
		cursorBin  = fs.String("cursor-bin", "cursor-agent", "cursor CLI binary path")
	)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *moduleID == "" || *modelName == "" {
		fmt.Fprintln(os.Stderr, "usage: ai_eval run --module m4 --model <MODEL_X> [--judge-model <JUDGE_MODEL>]")
		return 2
	}

	result, err := core.RunAutoEvaluation(context.Background(), core.AutoRunConfig{
		WorkspaceRoot: ".",
		CursorBin:     *cursorBin,
		Model:         *modelName,
		Module:        *moduleID,
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

func printUsage() {
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
