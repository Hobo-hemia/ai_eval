package eval

import (
	"regexp"
	"strings"
)

var invalidModelDirChars = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func ModelDirName(model string) string {
	trimmed := strings.TrimSpace(model)
	if trimmed == "" {
		return "unknown-model"
	}
	safe := invalidModelDirChars.ReplaceAllString(trimmed, "-")
	safe = strings.Trim(safe, "-._")
	if safe == "" {
		return "unknown-model"
	}
	return safe
}
