package module

import (
	"fmt"
	"slices"
	"strings"
)

var supportedModules = []string{
	"m1_arch",
	"m2_biz",
	"m3_component",
	"m4_bugfix",
}

func IsSupportedModule(module string) bool {
	return slices.Contains(supportedModules, module)
}

func SupportedModules() []string {
	return append([]string(nil), supportedModules...)
}

func DefaultDirectories() []string {
	return []string{
		"docs",
		"templates",
		"eval_records",
		"modules/m1_arch/input",
		"modules/m1_arch/tests",
		"modules/m2_biz/input",
		"modules/m2_biz/tests",
		"modules/m3_component/input",
		"modules/m3_component/tests",
		"modules/m4_bugfix/input",
		"modules/m4_bugfix/tests",
	}
}

func ResultFileByModule(module string) string {
	switch module {
	case "m1_arch":
		return "m1_result.proto"
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

func NormalizeAutoModule(raw string) (string, error) {
	switch normalizeModule(raw) {
	case "m1":
		return "m1_arch", nil
	case "m2":
		return "m2_biz", nil
	case "m3":
		return "m3_component", nil
	case "m4":
		return "m4_bugfix", nil
	default:
		return "", fmt.Errorf("unsupported module: %s (expected: m1/m1_arch or m2/m2_biz or m3/m3_component or m4/m4_bugfix)", raw)
	}
}

func normalizeModule(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "m1", "m1_arch":
		return "m1"
	case "m2", "m2_biz":
		return "m2"
	case "m3", "m3_component":
		return "m3"
	case "m4", "m4_bugfix":
		return "m4"
	default:
		return ""
	}
}
