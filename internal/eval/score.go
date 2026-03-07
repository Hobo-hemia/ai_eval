package eval

import (
	"encoding/json"
	"time"
)

type Score struct {
	ModuleEvaluated string                 `json:"module_evaluated"`
	Model           string                 `json:"model"`
	TotalScore      int                    `json:"total_score"`
	Breakdown       map[string]ScoreDetail `json:"breakdown"`
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
		},
		FinalReasoning: "",
		GeneratedAt:    now,
	}

	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(b)
}
