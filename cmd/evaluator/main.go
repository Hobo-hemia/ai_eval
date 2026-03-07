package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"ai_eval/internal/eval"
)

func main() {
	var (
		module = flag.String("module", "", "module id: m1_arch|m2_biz|m3_component|m4_bugfix")
		model  = flag.String("model", "", "model id: gemini-3.1|gpt-5.3|claude-opus-4.6|qwen-3.5|kimi-2.5")
		phase  = flag.String("phase", "prepare", "phase: prepare|record")
	)
	flag.Parse()

	ctx := context.Background()
	app := eval.NewApplication(eval.Config{
		WorkspaceRoot: ".",
		Now:           time.Now,
	})

	switch *phase {
	case "prepare":
		if err := app.Prepare(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "prepare failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("prepare success")
	case "record":
		if *module == "" || *model == "" {
			fmt.Fprintln(os.Stderr, "record phase requires -module and -model")
			os.Exit(1)
		}
		if err := app.InitRecord(ctx, *model, *module); err != nil {
			fmt.Fprintf(os.Stderr, "init record failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("record initialized")
	default:
		fmt.Fprintf(os.Stderr, "unsupported phase: %s\n", *phase)
		os.Exit(1)
	}
}
