package workflow

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"
)

type phase1Params struct {
	root       string
	moduleID   string
	modelName  string
	cursorBin  string
	resultFile string
}

func runPhase1(ctx context.Context, p phase1Params) (time.Duration, error) {
	workspace, cleanup, err := preparePhase1Workspace(p.root, p.moduleID)
	if err != nil {
		return 0, err
	}
	defer cleanup()

	prompt, err := phase1PromptByModule(p.moduleID)
	if err != nil {
		return 0, err
	}
	logProgress("Phase1", "Starting code generation via isolated workspace model=%s", p.modelName)
	startedAt := time.Now()
	out, err := runCursorPrompt(ctx, p.cursorBin, p.modelName, workspace, prompt, "Phase1")
	if err != nil {
		return 0, fmt.Errorf("phase1 cursor-agent failed: %w", err)
	}
	duration := time.Since(startedAt)
	logProgress("Phase1", "Cursor-agent response received (%d chars)", len(out))

	blocks := extractCodeBlocks(out)
	if len(blocks) == 0 {
		return 0, errors.New("phase1 output has no code block")
	}
	logProgress("Phase1", "Parsed %d code block(s)", len(blocks))
	if err := os.WriteFile(p.resultFile, []byte(blocks[0]), 0o644); err != nil {
		return 0, fmt.Errorf("write result file: %w", err)
	}
	logProgress("Phase1", "Wrote result file %s", p.resultFile)
	return duration, nil
}
