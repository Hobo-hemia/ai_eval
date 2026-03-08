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
	phase2Failed bool
	phase2Error  string
}

func runJudgePhase(ctx context.Context, p judgeParams) (time.Duration, error) {
	preRuntime := module.RuntimeMetrics{
		Phase1Seconds: p.phase1Second,
		Phase2Seconds: p.phase2Second,
		Phase3Seconds: 0,
		TotalSeconds:  p.phase1Second + p.phase2Second,
	}
	prompt, err := phase3PromptByModule(p.moduleID, p.modelDir, p.judgeModel, preRuntime, p.phase2Failed, p.phase2Error)
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
		logProgress("Phase3", "Judge JSON parse failed, fallback score will be written: %v", err)
		scoreJSON = fallbackScoreJSON(p, runtime, err)
	}
	if err := os.WriteFile(p.scoreFile, scoreJSON, 0o644); err != nil {
		return 0, fmt.Errorf("write score file: %w", err)
	}
	return duration, nil
}

func fallbackScoreJSON(p judgeParams, runtime module.RuntimeMetrics, parseErr error) []byte {
	compileScore := 10
	testScore := 10
	total := 40
	reason := fmt.Sprintf(
		"judge output parsing failed, fallback scoring applied: %v",
		parseErr,
	)
	if p.phase2Failed {
		compileScore = 0
		testScore = 0
		total = 20
		reason = fmt.Sprintf(
			"phase2 failed and judge output parsing failed, fallback scoring applied: phase2=%s; parse=%v",
			p.phase2Error,
			parseErr,
		)
	}
	s := module.Score{
		ModuleEvaluated: p.moduleID,
		Model:           p.modelName,
		JudgeModel:      p.judgeModel,
		TotalScore:      total,
		Breakdown: map[string]module.ScoreDetail{
			"execution_compile": {
				Dimension: "编译通过率",
				Score:     compileScore,
				MaxScore:  25,
			},
			"execution_test": {
				Dimension: "功能/测试通过率",
				Score:     testScore,
				MaxScore:  25,
			},
			"static_analysis": {
				Dimension: "代码规范与特定维度",
				Score:     10,
				MaxScore:  25,
			},
			"execution_runtime": {
				Dimension: "运行时效率",
				Score:     10,
				MaxScore:  25,
			},
		},
		RuntimeMetrics: runtime,
		FinalReasoning: reason,
		GeneratedAt:    time.Now(),
	}
	out, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return []byte(module.DefaultScoreJSON(p.modelName, p.moduleID, time.Now()))
	}
	return out
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
