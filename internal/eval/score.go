package eval

import (
	"encoding/json"
	"time"
)

type Score struct {
	ModuleEvaluated string                 `json:"module_evaluated"`
	Model           string                 `json:"model"`
	JudgeModel      string                 `json:"judge_model,omitempty"`
	TotalScore      int                    `json:"total_score"`
	Breakdown       map[string]ScoreDetail `json:"breakdown"`
	RuntimeMetrics  RuntimeMetrics         `json:"runtime_metrics"`
	FinalReasoning  string                 `json:"final_reasoning"`
	GeneratedAt     time.Time              `json:"generated_at"`
}

type ScoreDetail struct {
	Dimension    string `json:"dimension"`
	Score        int    `json:"score"`
	MaxScore     int    `json:"max_score"`
	LogEvidence  string `json:"log_evidence,omitempty"`
	CodeEvidence string `json:"code_evidence,omitempty"`
}

type RuntimeMetrics struct {
	Phase1Seconds float64 `json:"phase1_seconds"`
	Phase2Seconds float64 `json:"phase2_seconds"`
	Phase3Seconds float64 `json:"phase3_seconds"`
	TotalSeconds  float64 `json:"total_seconds"`
}

func DefaultScoreJSON(model, module string, now time.Time) string {
	s := Score{
		ModuleEvaluated: module,
		Model:           model,
		TotalScore:      0,
		Breakdown: map[string]ScoreDetail{
			"execution_compile": {
				Dimension: "编译通过率",
				Score:     0,
				MaxScore:  0,
			},
			"execution_test": {
				Dimension: "功能/测试通过率",
				Score:     0,
				MaxScore:  0,
			},
			"static_analysis": {
				Dimension: "代码规范与特定维度",
				Score:     0,
				MaxScore:  0,
			},
			"execution_runtime": {
				Dimension: "运行时效率",
				Score:     0,
				MaxScore:  0,
			},
		},
		RuntimeMetrics: RuntimeMetrics{},
		FinalReasoning: "",
		GeneratedAt:    now,
	}

	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(b)
}
