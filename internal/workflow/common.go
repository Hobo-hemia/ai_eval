package workflow

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

func runCursorPrompt(
	ctx context.Context,
	cursorBin, modelName, workDir, prompt, stage string,
) (string, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(
		timeoutCtx,
		cursorBin,
		"-f",
		"-p",
		"--output-format",
		"text",
		"--model",
		modelName,
		prompt,
	)
	cmd.Dir = workDir
	out, err := runCommandWithHeartbeat(timeoutCtx, cmd, 15*time.Second, stage)
	if err != nil {
		if strings.Contains(string(out), "Press any key to sign in") {
			return "", fmt.Errorf("%w (cursor-agent not authenticated, run `cursor-agent login` first)", err)
		}
		return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

func logProgress(stage, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("[%s] [%s] %s\n", time.Now().Format("15:04:05"), stage, msg)
}

func roundSeconds(d time.Duration) float64 {
	return float64(d.Round(100*time.Millisecond)) / float64(time.Second)
}
