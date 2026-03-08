package internal

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ai_eval/internal/module"
	"ai_eval/internal/workflow"
)

type Config struct {
	WorkspaceRoot string
	Now           func() time.Time
}

type Application struct {
	cfg Config
}

func NewApplication(cfg Config) *Application {
	if cfg.WorkspaceRoot == "" {
		cfg.WorkspaceRoot = "."
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &Application{cfg: cfg}
}

func (a *Application) Prepare(_ context.Context) error {
	for _, d := range module.DefaultDirectories() {
		abs := filepath.Join(a.cfg.WorkspaceRoot, d)
		if err := os.MkdirAll(abs, 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", abs, err)
		}
	}
	return nil
}

func (a *Application) InitRecord(_ context.Context, modelName, moduleID string) error {
	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		return errors.New("model is required")
	}
	if !module.IsSupportedModule(moduleID) {
		return fmt.Errorf("unsupported module: %s", moduleID)
	}

	recordDir := filepath.Join(a.cfg.WorkspaceRoot, "eval_records", module.ModelDirName(modelName), moduleID)
	if err := os.MkdirAll(recordDir, 0o755); err != nil {
		return fmt.Errorf("mkdir record dir: %w", err)
	}
	if err := WriteFileIfNotExist(filepath.Join(recordDir, module.ResultFileByModule(moduleID)), DefaultResultGo()); err != nil {
		return err
	}
	if err := WriteFileIfNotExist(filepath.Join(recordDir, module.BuildLogFileByModule(moduleID)), ""); err != nil {
		return err
	}
	if err := WriteFileIfNotExist(filepath.Join(recordDir, module.TestLogFileByModule(moduleID)), ""); err != nil {
		return err
	}
	if err := WriteFileIfNotExist(filepath.Join(recordDir, "score.json"), module.DefaultScoreJSON(modelName, moduleID, a.cfg.Now())); err != nil {
		return err
	}
	return nil
}

func WriteFileIfNotExist(path, content string) error {
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

func DefaultResultGo() string {
	return "package result\n"
}

type AutoRunConfig = workflow.AutoRunConfig
type AutoRunResult = workflow.AutoRunResult
type AutoRunBatchResult = workflow.AutoRunBatchResult

func RunAutoEvaluation(ctx context.Context, cfg AutoRunConfig) (*AutoRunResult, error) {
	return workflow.RunAutoEvaluation(ctx, cfg)
}

func RunAutoEvaluationBatch(ctx context.Context, cfg AutoRunConfig, modules []string) (*AutoRunBatchResult, error) {
	return workflow.RunAutoEvaluationBatch(ctx, cfg, modules)
}

func ModelDirName(modelName string) string {
	return module.ModelDirName(modelName)
}

func SupportedModules() []string {
	return module.SupportedModules()
}
