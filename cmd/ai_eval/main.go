package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"ai_eval/internal/eval"
)

func main() {
	var (
		module     = flag.String("module", "", "module id, currently supports: m4|m4_bugfix")
		model      = flag.String("model", "", "model id to evaluate")
		judgeModel = flag.String("judge-model", "opus-4.6", "judge model id")
		cursorBin  = flag.String("cursor-bin", "cursor-agent", "cursor CLI binary path")
	)
	flag.Parse()

	if *module == "" || *model == "" {
		fmt.Fprintln(os.Stderr, "usage: ai_eval --module m4 --model <MODEL_X> [--judge-model <JUDGE_MODEL>]")
		os.Exit(2)
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
		os.Exit(1)
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
}
