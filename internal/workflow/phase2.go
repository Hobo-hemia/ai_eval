package workflow

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

func runHarnessByModule(ctx context.Context, root, modelDir, moduleID string) error {
	switch moduleID {
	case "m1_arch":
		return runHarnessScript(ctx, root, "modules/m1_arch/tests/run_full_chain.sh", modelDir)
	case "m2_biz":
		return runHarnessScript(ctx, root, "modules/m2_biz/tests/run_full_chain.sh", modelDir)
	case "m3_component":
		return runHarnessScript(ctx, root, "modules/m3_component/tests/run_full_chain.sh", modelDir)
	case "m4_bugfix":
		return runHarnessScript(ctx, root, "modules/m4_bugfix/tests/run_full_chain.sh", modelDir)
	default:
		return fmt.Errorf("unsupported module for harness: %s", moduleID)
	}
}

func runHarnessScript(ctx context.Context, root, scriptPath, modelDir string) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(timeoutCtx, "bash", scriptPath, modelDir)
	cmd.Dir = root
	out, err := runCommandWithHeartbeat(timeoutCtx, cmd, 10*time.Second, "Phase2")
	if err != nil {
		return fmt.Errorf("run harness failed: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func runCommandWithHeartbeat(
	ctx context.Context,
	cmd *exec.Cmd,
	heartbeatInterval time.Duration,
	stage string,
) ([]byte, error) {
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	start := time.Now()
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case err := <-done:
			logProgress(stage, "Finished subprocess in %s", time.Since(start).Round(time.Second))
			return buf.Bytes(), err
		case <-ticker.C:
			logProgress(stage, "Still running... elapsed=%s", time.Since(start).Round(time.Second))
		case <-ctx.Done():
			return buf.Bytes(), ctx.Err()
		}
	}
}
