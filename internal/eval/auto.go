package eval

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type AutoRunConfig struct {
	WorkspaceRoot string
	CursorBin     string
	Model         string
	Module        string
	JudgeModel    string
}

type AutoRunResult struct {
	Model        string
	ModelDir     string
	Module       string
	JudgeModel   string
	ResultFile   string
	BuildLogFile string
	TestLogFile  string
	ScoreFile    string
}

func RunAutoEvaluation(ctx context.Context, cfg AutoRunConfig) (*AutoRunResult, error) {
	runStartedAt := time.Now()
	root := cfg.WorkspaceRoot
	if root == "" {
		root = "."
	}
	cursorBin := cfg.CursorBin
	if cursorBin == "" {
		cursorBin = "cursor-agent"
	}
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		return nil, errors.New("model is required")
	}

	moduleID, err := normalizeAutoModule(cfg.Module)
	if err != nil {
		return nil, err
	}
	if moduleID != "m4_bugfix" {
		return nil, fmt.Errorf("auto runner currently only supports m4/m4_bugfix")
	}
	logProgress("Init", "Resolved module=%s model=%s", moduleID, model)

	judgeModel := strings.TrimSpace(cfg.JudgeModel)
	if judgeModel == "" {
		judgeModel = "opus-4.6"
	}

	modelDir := ModelDirName(model)
	recordDir := filepath.Join(root, "eval_records", modelDir, moduleID)
	logProgress("Init", "Model directory=%s", modelDir)
	if err := os.MkdirAll(recordDir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir record dir: %w", err)
	}
	cleanupIntermediateArtifacts(recordDir)

	resultFile := filepath.Join(recordDir, ResultFileByModule(moduleID))
	buildLogFile := filepath.Join(recordDir, BuildLogFileByModule(moduleID))
	testLogFile := filepath.Join(recordDir, TestLogFileByModule(moduleID))
	scoreFile := filepath.Join(recordDir, "score.json")

	if err := WriteFileIfNotExist(resultFile, DefaultResultGo()); err != nil {
		return nil, err
	}
	_ = os.WriteFile(buildLogFile, []byte(""), 0o644)
	_ = os.WriteFile(testLogFile, []byte(""), 0o644)
	_ = os.WriteFile(scoreFile, []byte(DefaultScoreJSON(model, moduleID, time.Now())), 0o644)
	logProgress("Init", "Record files ready at %s", recordDir)

	moduleDir := filepath.Join(root, "modules", moduleID)
	phase1Prompt := m4Phase1Prompt()
	logProgress("Phase1", "Starting code generation via cursor-agent model=%s", model)
	phase1StartedAt := time.Now()
	phase1Out, err := runCursorPrompt(ctx, cursorBin, model, moduleDir, phase1Prompt, "Phase1")
	if err != nil {
		return nil, fmt.Errorf("phase1 cursor-agent failed: %w", err)
	}
	phase1Duration := time.Since(phase1StartedAt)
	logProgress("Phase1", "Cursor-agent response received (%d chars)", len(phase1Out))

	codeBlocks := extractCodeBlocks(phase1Out)
	if len(codeBlocks) == 0 {
		return nil, errors.New("phase1 output has no code block")
	}
	logProgress("Phase1", "Parsed %d code block(s)", len(codeBlocks))
	// 第一段是业务代码，强制落盘为 m4_result.go
	if err := os.WriteFile(resultFile, []byte(codeBlocks[0]), 0o644); err != nil {
		return nil, fmt.Errorf("write result file: %w", err)
	}
	logProgress("Phase1", "Wrote result file %s", resultFile)

	logProgress("Phase2", "Running harness build+test")
	phase2StartedAt := time.Now()
	if err := runM4Harness(ctx, root, modelDir); err != nil {
		return nil, err
	}
	phase2Duration := time.Since(phase2StartedAt)
	logProgress("Phase2", "Harness completed, logs written")

	preJudgeRuntime := RuntimeMetrics{
		Phase1Seconds: roundSeconds(phase1Duration),
		Phase2Seconds: roundSeconds(phase2Duration),
		Phase3Seconds: 0,
		TotalSeconds:  roundSeconds(phase1Duration + phase2Duration),
	}
	phase2Prompt := m4Phase3Prompt(modelDir, judgeModel, preJudgeRuntime)
	logProgress("Phase3", "Starting judge via cursor-agent model=%s", judgeModel)
	phase3StartedAt := time.Now()
	phase3Out, err := runCursorPrompt(ctx, cursorBin, judgeModel, root, phase2Prompt, "Phase3")
	if err != nil {
		return nil, fmt.Errorf("phase3 cursor-agent failed: %w", err)
	}
	phase3Duration := time.Since(phase3StartedAt)
	logProgress("Phase3", "Judge response received (%d chars)", len(phase3Out))

	runtime := RuntimeMetrics{
		Phase1Seconds: roundSeconds(phase1Duration),
		Phase2Seconds: roundSeconds(phase2Duration),
		Phase3Seconds: roundSeconds(phase3Duration),
		TotalSeconds:  roundSeconds(time.Since(runStartedAt)),
	}
	scoreJSON, err := normalizeJudgeJSON(phase3Out, model, moduleID, judgeModel, runtime)
	if err != nil {
		return nil, fmt.Errorf("parse phase2 json failed: %w", err)
	}
	if err := os.WriteFile(scoreFile, scoreJSON, 0o644); err != nil {
		return nil, fmt.Errorf("write score file: %w", err)
	}
	cleanupIntermediateArtifacts(recordDir)
	logProgress("Done", "Completed all stages in %s", time.Since(runStartedAt).Round(time.Second))

	return &AutoRunResult{
		Model:        model,
		ModelDir:     modelDir,
		Module:       moduleID,
		JudgeModel:   judgeModel,
		ResultFile:   resultFile,
		BuildLogFile: buildLogFile,
		TestLogFile:  testLogFile,
		ScoreFile:    scoreFile,
	}, nil
}

func normalizeAutoModule(raw string) (string, error) {
	module := strings.TrimSpace(strings.ToLower(raw))
	switch module {
	case "m4", "m4_bugfix":
		return "m4_bugfix", nil
	default:
		return "", fmt.Errorf("unsupported module: %s (expected: m4 or m4_bugfix)", raw)
	}
}

func runCursorPrompt(
	ctx context.Context,
	cursorBin, model, workDir, prompt, stage string,
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
		model,
		prompt,
	)
	cmd.Dir = workDir
	out, err := runCommandWithHeartbeat(timeoutCtx, cmd, 15*time.Second, stage)
	if err != nil {
		if strings.Contains(string(out), "Press any key to sign in") {
			return "", fmt.Errorf(
				"%w (cursor-agent not authenticated, run `cursor-agent login` first)",
				err,
			)
		}
		return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

func runM4Harness(ctx context.Context, root, model string) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(
		timeoutCtx,
		"bash",
		"modules/m4_bugfix/tests/run_full_chain.sh",
		model,
	)
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
			elapsed := time.Since(start).Round(time.Second)
			logProgress(stage, "Still running... elapsed=%s", elapsed)
		case <-ctx.Done():
			return buf.Bytes(), ctx.Err()
		}
	}
}

func logProgress(stage, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("[%s] [%s] %s\n", time.Now().Format("15:04:05"), stage, msg)
}

func cleanupIntermediateArtifacts(recordDir string) {
	paths := []string{
		filepath.Join(recordDir, "phase1_raw_output.txt"),
		filepath.Join(recordDir, "phase2_raw_output.txt"),
		filepath.Join(recordDir, "m4_generated_test.go"),
		filepath.Join(recordDir, "m4_result_test.go"),
	}
	for _, p := range paths {
		_ = os.Remove(p)
	}
}

func roundSeconds(d time.Duration) float64 {
	return float64(d.Round(100*time.Millisecond)) / float64(time.Second)
}

func m4Phase1Prompt() string {
	return strings.TrimSpace(`
你现在在 m4_bugfix 模块评测中，请严格按以下文件执行，不要省略任何要求：
- @input/guidance.md
- @input/bug_report.md
- @input/legacy_code.go
- @.cursorrules

硬性要求：
1) 按 TDD 思路修复 legacy_code.go 中缺陷。
2) 必须保留对外签名不变。
3) 必须在关键修复处加注释：// BUGFIX: [根因与修复逻辑]
4) 测试必须使用 table-driven + gomock + testify/assert。
5) 只输出两段代码块，不要输出解释文字：
   - 第一段：修复后的业务代码（package result）
   - 第二段：对应 _test.go 测试代码（package result）
`)
}

func m4Phase3Prompt(modelDir, judgeModel string, runtime RuntimeMetrics) string {
	return strings.TrimSpace(fmt.Sprintf(`
请作为 M4 裁判严格评分。你必须遵循以下规则文件：
- @templates/phase2_judge_prompt.md
- @modules/m4_bugfix/JUDGE_AGENT.md

评分输入材料：
- @eval_records/%[1]s/m4_bugfix/m4_result.go
- @eval_records/%[1]s/m4_bugfix/m4_build.log
- @eval_records/%[1]s/m4_bugfix/m4_test.log

运行时长指标（必须在输出 JSON 中填写 runtime_metrics）：
- phase1_seconds = %[3]0.1f
- phase2_seconds = %[4]0.1f
- phase3_seconds = %[5]0.1f
- total_seconds = %[6]0.1f

输出要求（强制）：
1) 只输出 JSON，禁止 markdown 和额外说明
2) 按 100 分制给出总分与分项
3) 必须检查是否存在 // BUGFIX: 注释，否则 D3 记 0
4) 在 JSON 中填写：
   - "judge_model": "%[2]s"
   - "runtime_metrics": {"phase1_seconds": x, "phase2_seconds": y, "phase3_seconds": z, "total_seconds": t}
`, modelDir, judgeModel, runtime.Phase1Seconds, runtime.Phase2Seconds, runtime.Phase3Seconds, runtime.TotalSeconds))
}

func extractCodeBlocks(output string) []string {
	re := regexp.MustCompile("(?s)```(?:[a-zA-Z0-9_+-]+)?\\n(.*?)```")
	matches := re.FindAllStringSubmatch(output, -1)
	blocks := make([]string, 0, len(matches))
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		blocks = append(blocks, strings.TrimSpace(m[1])+"\n")
	}
	return blocks
}

func normalizeJudgeJSON(
	raw, model, module, judgeModel string,
	runtime RuntimeMetrics,
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
		obj["model"] = model
	}
	if _, ok := obj["module_evaluated"]; !ok {
		obj["module_evaluated"] = module
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
