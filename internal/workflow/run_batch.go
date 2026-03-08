package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"ai_eval/internal/module"
)

func RunAutoEvaluationBatch(ctx context.Context, cfg AutoRunConfig, modules []string) (*AutoRunBatchResult, error) {
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
	judgeModel := strings.TrimSpace(cfg.JudgeModel)
	if judgeModel == "" {
		judgeModel = "gemini-3-flash"
	}
	normModules, err := normalizeBatchModules(modules)
	if err != nil {
		return nil, err
	}
	modelDir := module.ModelDirName(modelName)
	logProgress("Init", "Resolved batch modules=%s model=%s", strings.Join(normModules, ","), modelName)

	resultFiles := map[string]string{}
	buildLogFiles := map[string]string{}
	testLogFiles := map[string]string{}
	scoreFiles := map[string]string{}
	for _, moduleID := range normModules {
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
		resultFiles[moduleID] = resultFile
		buildLogFiles[moduleID] = buildLogFile
		testLogFiles[moduleID] = testLogFile
		scoreFiles[moduleID] = scoreFile
	}

	phase1StartedAt := time.Now()
	workspace, cleanup, err := preparePhase1WorkspaceForModules(root, normModules)
	if err != nil {
		return nil, err
	}
	defer cleanup()
	phase1Prompt, err := phase1PromptForModules(normModules)
	if err != nil {
		return nil, err
	}
	logProgress("Phase1", "Starting batch code generation via isolated workspace model=%s", modelName)
	phase1Raw, err := runCursorPrompt(ctx, cursorBin, modelName, workspace, phase1Prompt, "Phase1")
	if err != nil {
		return nil, fmt.Errorf("phase1 cursor-agent failed: %w", err)
	}
	phase1Duration := time.Since(phase1StartedAt)
	tagBlocks := extractTaggedCodeBlocks(phase1Raw)
	resultContent := map[string]string{}
	for _, moduleID := range normModules {
		tag := phase1TagByModule(moduleID)
		content, ok := tagBlocks[tag]
		if !ok {
			logProgress("Phase1", "Missing tagged block %s, write default content for %s", tag, moduleID)
			content = defaultResultByModule(moduleID)
		}
		if err := os.WriteFile(resultFiles[moduleID], []byte(content), 0o644); err != nil {
			return nil, fmt.Errorf("write batch result file %s: %w", moduleID, err)
		}
		resultContent[moduleID] = content
		logProgress("Phase1", "Wrote result file %s", resultFiles[moduleID])
	}
	phase1ByModule := allocatePhase1BySize(normModules, phase1Duration, resultContent)

	phase2Durations := map[string]time.Duration{}
	phase2Errors := map[string]error{}
	for _, moduleID := range normModules {
		startedAt := time.Now()
		logProgress("Phase2", "Running harness build+test for %s", moduleID)
		err := runHarnessByModule(ctx, root, modelDir, moduleID)
		phase2Durations[moduleID] = time.Since(startedAt)
		phase2Errors[moduleID] = err
		if err != nil {
			logProgress("Phase2", "Harness failed for %s, continue to judge with penalty: %v", moduleID, err)
		} else {
			logProgress("Phase2", "Harness completed for %s", moduleID)
		}
	}

	phase3Duration, scoreByModule, err := runUnifiedJudgePhase(ctx, unifiedJudgeParams{
		root:            root,
		cursorBin:       cursorBin,
		modelName:       modelName,
		modelDir:        modelDir,
		judgeModel:      judgeModel,
		modules:         normModules,
		phase1ByModule:  phase1ByModule,
		phase2Durations: phase2Durations,
		phase2Errors:    phase2Errors,
	})
	if err != nil {
		return nil, err
	}
	for _, moduleID := range normModules {
		score, ok := scoreByModule[moduleID]
		if !ok {
			score = fallbackScoreForModule(modelName, judgeModel, moduleID, phase1ByModule[moduleID], roundSeconds(phase2Durations[moduleID]), roundSeconds(phase3Duration), phase2Errors[moduleID], errors.New("missing module score from unified judge"))
		}
		out, err := json.MarshalIndent(score, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("marshal score for %s: %w", moduleID, err)
		}
		if err := os.WriteFile(scoreFiles[moduleID], out, 0o644); err != nil {
			return nil, fmt.Errorf("write score file %s: %w", moduleID, err)
		}
		recordDir := filepath.Dir(scoreFiles[moduleID])
		cleanupIntermediateArtifacts(recordDir, moduleID)
	}

	logProgress("Done", "Completed batch stages in %s", time.Since(runStartedAt).Round(time.Second))
	return &AutoRunBatchResult{
		Model:      modelName,
		ModelDir:   modelDir,
		JudgeModel: judgeModel,
		Modules:    normModules,
		ScoreFiles: scoreFiles,
	}, nil
}

type unifiedJudgeParams struct {
	root            string
	cursorBin       string
	modelName       string
	modelDir        string
	judgeModel      string
	modules         []string
	phase1ByModule  map[string]float64
	phase2Durations map[string]time.Duration
	phase2Errors    map[string]error
}

func runUnifiedJudgePhase(ctx context.Context, p unifiedJudgeParams) (time.Duration, map[string]module.Score, error) {
	phase2Status := map[string]string{}
	phase2Error := map[string]string{}
	runtimeMap := map[string]module.RuntimeMetrics{}
	for _, m := range p.modules {
		if p.phase2Errors[m] == nil {
			phase2Status[m] = "pass"
			phase2Error[m] = ""
		} else {
			phase2Status[m] = "failed"
			phase2Error[m] = p.phase2Errors[m].Error()
		}
		runtimeMap[m] = module.RuntimeMetrics{
			Phase1Seconds: p.phase1ByModule[m],
			Phase2Seconds: roundSeconds(p.phase2Durations[m]),
			Phase3Seconds: 0,
			TotalSeconds:  p.phase1ByModule[m] + roundSeconds(p.phase2Durations[m]),
		}
	}
	prompt, err := phase3PromptForModules(p.modules, p.modelDir, p.judgeModel, phase2Status, phase2Error, runtimeMap)
	if err != nil {
		return 0, nil, err
	}
	logProgress("Phase3", "Starting unified judge via cursor-agent model=%s", p.judgeModel)
	startedAt := time.Now()
	raw, err := runCursorPrompt(ctx, p.cursorBin, p.judgeModel, p.root, prompt, "Phase3")
	if err != nil {
		return 0, nil, fmt.Errorf("phase3 cursor-agent failed: %w", err)
	}
	duration := time.Since(startedAt)
	logProgress("Phase3", "Unified judge response received (%d chars)", len(raw))
	scoreMap := map[string]module.Score{}
	payload, err := extractJSONObject(raw)
	if err != nil {
		logProgress("Phase3", "Unified judge json parse failed, fallback for all modules: %v", err)
		return duration, fallbackScoreMap(p, duration, err), nil
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(payload), &obj); err != nil {
		logProgress("Phase3", "Unified judge json unmarshal failed, fallback for all modules: %v", err)
		return duration, fallbackScoreMap(p, duration, err), nil
	}
	moduleScoresRaw, ok := obj["module_scores"].(map[string]any)
	if !ok {
		logProgress("Phase3", "Unified judge missing module_scores, fallback for all modules")
		return duration, fallbackScoreMap(p, duration, errors.New("missing module_scores")), nil
	}
	for _, m := range p.modules {
		moduleRaw, ok := moduleScoresRaw[m]
		if !ok {
			scoreMap[m] = fallbackScoreForModule(p.modelName, p.judgeModel, m, p.phase1ByModule[m], roundSeconds(p.phase2Durations[m]), roundSeconds(duration), p.phase2Errors[m], fmt.Errorf("missing module score: %s", m))
			continue
		}
		blockBytes, err := json.Marshal(moduleRaw)
		if err != nil {
			scoreMap[m] = fallbackScoreForModule(p.modelName, p.judgeModel, m, p.phase1ByModule[m], roundSeconds(p.phase2Durations[m]), roundSeconds(duration), p.phase2Errors[m], err)
			continue
		}
		normalized, err := normalizeJudgeJSON(
			string(blockBytes),
			p.modelName,
			m,
			p.judgeModel,
			module.RuntimeMetrics{
				Phase1Seconds: p.phase1ByModule[m],
				Phase2Seconds: roundSeconds(p.phase2Durations[m]),
				Phase3Seconds: roundSeconds(duration),
				TotalSeconds:  p.phase1ByModule[m] + roundSeconds(p.phase2Durations[m]) + roundSeconds(duration),
			},
		)
		if err != nil {
			scoreMap[m] = fallbackScoreForModule(p.modelName, p.judgeModel, m, p.phase1ByModule[m], roundSeconds(p.phase2Durations[m]), roundSeconds(duration), p.phase2Errors[m], err)
			continue
		}
		var score module.Score
		if err := json.Unmarshal(normalized, &score); err != nil {
			score = fallbackScoreForModule(p.modelName, p.judgeModel, m, p.phase1ByModule[m], roundSeconds(p.phase2Durations[m]), roundSeconds(duration), p.phase2Errors[m], err)
		}
		scoreMap[m] = score
	}
	return duration, scoreMap, nil
}

func fallbackScoreMap(p unifiedJudgeParams, phase3Duration time.Duration, parseErr error) map[string]module.Score {
	out := map[string]module.Score{}
	for _, m := range p.modules {
		out[m] = fallbackScoreForModule(
			p.modelName,
			p.judgeModel,
			m,
			p.phase1ByModule[m],
			roundSeconds(p.phase2Durations[m]),
			roundSeconds(phase3Duration),
			p.phase2Errors[m],
			parseErr,
		)
	}
	return out
}

func fallbackScoreForModule(
	modelName, judgeModel, moduleID string,
	phase1, phase2, phase3 float64,
	phase2Err error,
	parseErr error,
) module.Score {
	compileScore := 10
	testScore := 10
	total := 40
	reason := fmt.Sprintf("fallback scoring applied: %v", parseErr)
	if phase2Err != nil {
		compileScore = 0
		testScore = 0
		total = 20
		reason = fmt.Sprintf("phase2 failed and fallback scoring applied: phase2=%v; parse=%v", phase2Err, parseErr)
	}
	return module.Score{
		ModuleEvaluated: moduleID,
		Model:           modelName,
		JudgeModel:      judgeModel,
		TotalScore:      total,
		Breakdown: map[string]module.ScoreDetail{
			"execution_compile": {Dimension: "编译通过率", Score: compileScore, MaxScore: 25},
			"execution_test":    {Dimension: "功能/测试通过率", Score: testScore, MaxScore: 25},
			"static_analysis":   {Dimension: "代码规范与特定维度", Score: 10, MaxScore: 25},
			"execution_runtime": {Dimension: "运行时效率", Score: 10, MaxScore: 25},
		},
		RuntimeMetrics: module.RuntimeMetrics{
			Phase1Seconds: phase1,
			Phase2Seconds: phase2,
			Phase3Seconds: phase3,
			TotalSeconds:  phase1 + phase2 + phase3,
		},
		FinalReasoning: reason,
		GeneratedAt:    time.Now(),
	}
}

func phase1TagByModule(moduleID string) string {
	switch moduleID {
	case "m1_arch":
		return "m1_proto"
	case "m2_biz":
		return "m2_go"
	case "m3_component":
		return "m3_go"
	case "m4_bugfix":
		return "m4_go"
	default:
		return ""
	}
}

func normalizeBatchModules(modules []string) ([]string, error) {
	if len(modules) == 0 {
		return nil, errors.New("empty modules")
	}
	out := make([]string, 0, len(modules))
	seen := map[string]struct{}{}
	for _, raw := range modules {
		mapped, err := module.NormalizeAutoModule(raw)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[mapped]; ok {
			continue
		}
		seen[mapped] = struct{}{}
		out = append(out, mapped)
	}
	slices.Sort(out)
	return out, nil
}

func allocatePhase1BySize(modules []string, total time.Duration, content map[string]string) map[string]float64 {
	out := map[string]float64{}
	if len(modules) == 0 {
		return out
	}
	totalSeconds := roundSeconds(total)
	if totalSeconds <= 0 {
		for _, m := range modules {
			out[m] = 0
		}
		return out
	}
	weights := map[string]float64{}
	var weightSum float64
	for _, m := range modules {
		w := float64(len(strings.TrimSpace(content[m])))
		if w <= 0 {
			w = 1
		}
		weights[m] = w
		weightSum += w
	}
	if weightSum <= 0 {
		each := totalSeconds / float64(len(modules))
		for _, m := range modules {
			out[m] = roundTo1(each)
		}
		return out
	}
	remaining := totalSeconds
	for idx, m := range modules {
		if idx == len(modules)-1 {
			out[m] = roundTo1(remaining)
			break
		}
		sec := roundTo1(totalSeconds * (weights[m] / weightSum))
		if sec < 0.1 {
			sec = 0.1
		}
		out[m] = sec
		remaining -= sec
	}
	return out
}

func roundTo1(v float64) float64 {
	return float64(int(v*10+0.5)) / 10
}
