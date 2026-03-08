package workflow

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ai_eval/internal/module"
)

func RunAutoEvaluation(ctx context.Context, cfg AutoRunConfig) (*AutoRunResult, error) {
	runStartedAt := time.Now()
	root := strings.TrimSpace(cfg.WorkspaceRoot)
	if root == "" {
		root = "."
	}
	cursorBin := strings.TrimSpace(cfg.CursorBin)
	if cursorBin == "" {
		cursorBin = "cursor-agent"
	}
	modelName := strings.TrimSpace(cfg.Model)
	if modelName == "" {
		return nil, errors.New("model is required")
	}
	moduleID, err := module.NormalizeAutoModule(cfg.Module)
	if err != nil {
		return nil, err
	}
	judgeModel := strings.TrimSpace(cfg.JudgeModel)
	if judgeModel == "" {
		judgeModel = "gemini-3-flash"
	}

	logProgress("Init", "Resolved module=%s model=%s", moduleID, modelName)
	modelDir := module.ModelDirName(modelName)
	recordDir := filepath.Join(root, "eval_records", modelDir, moduleID)
	if err := os.MkdirAll(recordDir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir record dir: %w", err)
	}
	cleanupIntermediateArtifacts(recordDir, moduleID)

	resultFile := filepath.Join(recordDir, module.ResultFileByModule(moduleID))
	buildLogFile := filepath.Join(recordDir, module.BuildLogFileByModule(moduleID))
	testLogFile := filepath.Join(recordDir, module.TestLogFileByModule(moduleID))
	scoreFile := filepath.Join(recordDir, "score.json")
	if err := writeRecordShell(resultFile, buildLogFile, testLogFile, scoreFile, modelName, moduleID); err != nil {
		return nil, err
	}

	phase1Duration, err := runPhase1(ctx, phase1Params{
		root:       root,
		moduleID:   moduleID,
		modelName:  modelName,
		cursorBin:  cursorBin,
		resultFile: resultFile,
	})
	if err != nil {
		return nil, err
	}

	phase2StartedAt := time.Now()
	logProgress("Phase2", "Running harness build+test")
	if err := runHarnessByModule(ctx, root, modelDir, moduleID); err != nil {
		return nil, err
	}
	phase2Duration := time.Since(phase2StartedAt)
	logProgress("Phase2", "Harness completed, logs written")

	if _, err := runJudgePhase(ctx, judgeParams{
		root:         root,
		moduleID:     moduleID,
		modelDir:     modelDir,
		modelName:    modelName,
		judgeModel:   judgeModel,
		cursorBin:    cursorBin,
		scoreFile:    scoreFile,
		phase1Second: roundSeconds(phase1Duration),
		phase2Second: roundSeconds(phase2Duration),
	}); err != nil {
		return nil, err
	}

	cleanupIntermediateArtifacts(recordDir, moduleID)
	logProgress("Done", "Completed all stages in %s", time.Since(runStartedAt).Round(time.Second))
	return &AutoRunResult{
		Model:        modelName,
		ModelDir:     modelDir,
		Module:       moduleID,
		JudgeModel:   judgeModel,
		ResultFile:   resultFile,
		BuildLogFile: buildLogFile,
		TestLogFile:  testLogFile,
		ScoreFile:    scoreFile,
	}, nil
}

func writeRecordShell(resultFile, buildLogFile, testLogFile, scoreFile, modelName, moduleID string) error {
	if err := writeFileIfNotExist(resultFile, defaultResultGo()); err != nil {
		return err
	}
	_ = os.WriteFile(buildLogFile, []byte(""), 0o644)
	_ = os.WriteFile(testLogFile, []byte(""), 0o644)
	_ = os.WriteFile(scoreFile, []byte(module.DefaultScoreJSON(modelName, moduleID, time.Now())), 0o644)
	return nil
}

func writeFileIfNotExist(path, content string) error {
	_, err := os.Stat(path)
	if err == nil {
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat %s: %w", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func defaultResultGo() string {
	return "package result\n"
}
