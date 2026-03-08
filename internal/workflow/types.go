package workflow

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
