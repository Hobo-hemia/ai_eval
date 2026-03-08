package workflow

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

func preparePhase1Workspace(root, moduleID string) (string, func(), error) {
	baseDir := filepath.Join(root, ".tmp")
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return "", nil, fmt.Errorf("mkdir tmp root failed: %w", err)
	}
	workspaceDir, err := os.MkdirTemp(baseDir, "ai_eval_phase1_"+moduleID+".")
	if err != nil {
		return "", nil, fmt.Errorf("create phase1 workspace failed: %w", err)
	}
	cleanup := func() {
		_ = os.RemoveAll(workspaceDir)
	}

	moduleDir := filepath.Join(root, "modules", moduleID)
	inputSrc := filepath.Join(moduleDir, "input")
	inputDst := filepath.Join(workspaceDir, "input")
	if err := copyDirectory(inputSrc, inputDst); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("copy input materials failed: %w", err)
	}

	ruleSrc := filepath.Join(moduleDir, ".cursorrules")
	if _, err := os.Stat(ruleSrc); err == nil {
		raw, readErr := os.ReadFile(ruleSrc)
		if readErr != nil {
			cleanup()
			return "", nil, fmt.Errorf("read .cursorrules failed: %w", readErr)
		}
		if writeErr := os.WriteFile(filepath.Join(workspaceDir, ".cursorrules"), raw, 0o644); writeErr != nil {
			cleanup()
			return "", nil, fmt.Errorf("write .cursorrules failed: %w", writeErr)
		}
	}
	return workspaceDir, cleanup, nil
}

func preparePhase1WorkspaceForModules(root string, modules []string) (string, func(), error) {
	baseDir := filepath.Join(root, ".tmp")
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return "", nil, fmt.Errorf("mkdir tmp root failed: %w", err)
	}
	workspaceDir, err := os.MkdirTemp(baseDir, "ai_eval_phase1_batch.")
	if err != nil {
		return "", nil, fmt.Errorf("create phase1 batch workspace failed: %w", err)
	}
	cleanup := func() {
		_ = os.RemoveAll(workspaceDir)
	}

	for _, moduleID := range modules {
		moduleDir := filepath.Join(root, "modules", moduleID)
		inputSrc := filepath.Join(moduleDir, "input")
		inputDst := filepath.Join(workspaceDir, "modules", moduleID, "input")
		if err := copyDirectory(inputSrc, inputDst); err != nil {
			cleanup()
			return "", nil, fmt.Errorf("copy input materials failed for %s: %w", moduleID, err)
		}
		ruleSrc := filepath.Join(moduleDir, ".cursorrules")
		if _, err := os.Stat(ruleSrc); err == nil {
			raw, readErr := os.ReadFile(ruleSrc)
			if readErr != nil {
				cleanup()
				return "", nil, fmt.Errorf("read .cursorrules failed for %s: %w", moduleID, readErr)
			}
			ruleDst := filepath.Join(workspaceDir, "modules", moduleID, ".cursorrules")
			if writeErr := os.WriteFile(ruleDst, raw, 0o644); writeErr != nil {
				cleanup()
				return "", nil, fmt.Errorf("write .cursorrules failed for %s: %w", moduleID, writeErr)
			}
		}
	}
	return workspaceDir, cleanup, nil
}

func copyDirectory(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, raw, 0o644)
	})
}

func cleanupIntermediateArtifacts(recordDir, moduleID string) {
	paths := []string{
		filepath.Join(recordDir, "phase1_raw_output.txt"),
		filepath.Join(recordDir, "phase2_raw_output.txt"),
	}
	switch moduleID {
	case "m1_arch":
		paths = append(paths,
			filepath.Join(recordDir, "m1_generated_test.go"),
			filepath.Join(recordDir, "m1_result_test.go"),
		)
	case "m2_biz":
		paths = append(paths,
			filepath.Join(recordDir, "m2_generated_test.go"),
			filepath.Join(recordDir, "m2_result_test.go"),
		)
	case "m3_component":
		paths = append(paths,
			filepath.Join(recordDir, "m3_generated_test.go"),
			filepath.Join(recordDir, "m3_result_test.go"),
		)
	case "m4_bugfix":
		paths = append(paths,
			filepath.Join(recordDir, "m4_generated_test.go"),
			filepath.Join(recordDir, "m4_result_test.go"),
		)
	}
	for _, p := range paths {
		_ = os.Remove(p)
	}
}
