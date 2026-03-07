package eval

import "slices"

var supportedModules = []string{
	"m1_arch",
	"m2_biz",
	"m3_component",
	"m4_bugfix",
}

var supportedModels = []string{
	"gemini-3.1",
	"gpt-5.3",
	"claude-opus-4.6",
	"qwen-3.5",
	"kimi-2.5",
}

func IsSupportedModule(module string) bool {
	return slices.Contains(supportedModules, module)
}

func IsSupportedModel(model string) bool {
	return slices.Contains(supportedModels, model)
}

func SupportedModules() []string {
	return append([]string(nil), supportedModules...)
}

func SupportedModels() []string {
	return append([]string(nil), supportedModels...)
}

func DefaultDirectories() []string {
	dirs := []string{
		"docs",
		"templates",
		"modules/m1_arch/input",
		"modules/m1_arch/tests",
		"modules/m2_biz/input",
		"modules/m2_biz/tests",
		"modules/m3_component/input",
		"modules/m3_component/tests",
		"modules/m4_bugfix/input",
		"modules/m4_bugfix/tests",
	}

	for _, model := range supportedModels {
		for _, module := range supportedModules {
			dirs = append(dirs, "eval_records/"+model+"/"+module)
		}
	}
	return dirs
}

func ResultFileByModule(module string) string {
	switch module {
	case "m1_arch":
		return "m1_result.go"
	case "m2_biz":
		return "m2_result.go"
	case "m3_component":
		return "m3_result.go"
	case "m4_bugfix":
		return "m4_result.go"
	default:
		return "result.go"
	}
}

func BuildLogFileByModule(module string) string {
	switch module {
	case "m1_arch":
		return "m1_build.log"
	case "m2_biz":
		return "m2_build.log"
	case "m3_component":
		return "m3_build.log"
	case "m4_bugfix":
		return "m4_build.log"
	default:
		return "build.log"
	}
}

func TestLogFileByModule(module string) string {
	switch module {
	case "m1_arch":
		return "m1_test.log"
	case "m2_biz":
		return "m2_test.log"
	case "m3_component":
		return "m3_test.log"
	case "m4_bugfix":
		return "m4_test.log"
	default:
		return "test.log"
	}
}
