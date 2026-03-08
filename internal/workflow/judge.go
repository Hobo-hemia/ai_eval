package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"ai_eval/internal/module"
)

type judgeParams struct {
	root         string
	moduleID     string
	modelDir     string
	modelName    string
	judgeModel   string
	cursorBin    string
	scoreFile    string
	phase1Second float64
	phase2Second float64
}

func runJudgePhase(ctx context.Context, p judgeParams) (time.Duration, error) {
	preRuntime := module.RuntimeMetrics{
		Phase1Seconds: p.phase1Second,
		Phase2Seconds: p.phase2Second,
		Phase3Seconds: 0,
		TotalSeconds:  p.phase1Second + p.phase2Second,
	}
	prompt, err := phase3PromptByModule(p.moduleID, p.modelDir, p.judgeModel, preRuntime)
	if err != nil {
		return 0, err
	}
	logProgress("Phase3", "Starting judge via cursor-agent model=%s", p.judgeModel)
	startedAt := time.Now()
	raw, err := runCursorPrompt(ctx, p.cursorBin, p.judgeModel, p.root, prompt, "Phase3")
	if err != nil {
		return 0, fmt.Errorf("phase3 cursor-agent failed: %w", err)
	}
	duration := time.Since(startedAt)
	logProgress("Phase3", "Judge response received (%d chars)", len(raw))

	runtime := module.RuntimeMetrics{
		Phase1Seconds: p.phase1Second,
		Phase2Seconds: p.phase2Second,
		Phase3Seconds: roundSeconds(duration),
		TotalSeconds:  p.phase1Second + p.phase2Second + roundSeconds(duration),
	}
	scoreJSON, err := normalizeJudgeJSON(raw, p.modelName, p.moduleID, p.judgeModel, runtime)
	if err != nil {
		return 0, fmt.Errorf("parse phase2 json failed: %w", err)
	}
	if err := os.WriteFile(p.scoreFile, scoreJSON, 0o644); err != nil {
		return 0, fmt.Errorf("write score file: %w", err)
	}
	return duration, nil
}

func normalizeJudgeJSON(
	raw, modelName, moduleID, judgeModel string,
	runtime module.RuntimeMetrics,
) ([]byte, error) {
	payload, err := extractJSONObject(raw)
	if err != nil {
		return nil, err
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(payload), &obj); err != nil {
		return nil, err
	}
	if _, ok := obj["model"]; !ok {
		obj["model"] = modelName
	}
	if _, ok := obj["module_evaluated"]; !ok {
		obj["module_evaluated"] = moduleID
	}
	if _, ok := obj["generated_at"]; !ok {
		obj["generated_at"] = time.Now().Format(time.RFC3339)
	}
	obj["judge_model"] = judgeModel
	obj["runtime_metrics"] = runtime
	return json.MarshalIndent(obj, "", "  ")
}

func extractJSONObject(s string) (string, error) {
	start := -1
	depth := 0
	inString := false
	escape := false
	for i, r := range s {
		if start == -1 {
			if r == '{' {
				start = i
				depth = 1
			}
			continue
		}
		if inString {
			if escape {
				escape = false
				continue
			}
			if r == '\\' {
				escape = true
				continue
			}
			if r == '"' {
				inString = false
			}
			continue
		}
		switch r {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return strings.TrimSpace(s[start : i+1]), nil
			}
		}
	}
	return "", errors.New("no complete JSON object found")
}
