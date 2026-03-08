package eval

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
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
	dirs := DefaultDirectories()
	for _, d := range dirs {
		abs := filepath.Join(a.cfg.WorkspaceRoot, d)
		if err := os.MkdirAll(abs, 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", abs, err)
		}
	}
	return nil
}

func (a *Application) InitRecord(_ context.Context, model, module string) error {
	model = strings.TrimSpace(model)
	if model == "" {
		return errors.New("model is required")
	}
	if !IsSupportedModule(module) {
		return fmt.Errorf("unsupported module: %s", module)
	}

	recordDir := filepath.Join(a.cfg.WorkspaceRoot, "eval_records", ModelDirName(model), module)
	if err := os.MkdirAll(recordDir, 0o755); err != nil {
		return fmt.Errorf("mkdir record dir: %w", err)
	}

	if err := WriteFileIfNotExist(filepath.Join(recordDir, ResultFileByModule(module)), DefaultResultGo()); err != nil {
		return err
	}
	if err := WriteFileIfNotExist(filepath.Join(recordDir, BuildLogFileByModule(module)), ""); err != nil {
		return err
	}
	if err := WriteFileIfNotExist(filepath.Join(recordDir, TestLogFileByModule(module)), ""); err != nil {
		return err
	}
	if err := WriteFileIfNotExist(filepath.Join(recordDir, "score.json"), DefaultScoreJSON(model, module, a.cfg.Now())); err != nil {
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
