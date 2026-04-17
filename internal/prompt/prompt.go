package prompt

import "strings"

// Vars holds the values for template expansion.
type Vars struct {
	BranchName string
	Author     string
}

// expandVars replaces known {{...}} placeholders in template with values from vars.
// Unknown tokens are left as-is.
func expandVars(template string, vars Vars) string {
	r := strings.NewReplacer(
		"{{branch_name}}", vars.BranchName,
		"{{author}}", vars.Author,
	)
	return r.Replace(template)
}

// estimateTokens returns a conservative token count estimate for text.
func estimateTokens(text string) int {
	return len(text) / 4
}

// Build expands template variables, assembles prompt + diff, and truncates
// the diff if the total exceeds maxTokens. A maxTokens of 0 means unlimited.
func Build(promptTemplate string, diff string, vars Vars, maxTokens int) (string, bool) {
	expanded := expandVars(promptTemplate, vars)

	if maxTokens == 0 {
		return expanded + "\n\n" + diff, false
	}

	promptTokens := estimateTokens(expanded)
	if promptTokens >= maxTokens {
		return expanded, true
	}

	full := expanded + "\n\n" + diff
	if estimateTokens(full) <= maxTokens {
		return full, false
	}

	remaining := (maxTokens - promptTokens) * 4
	// Account for the separator
	remaining -= len("\n\n")
	if remaining < 0 {
		remaining = 0
	}
	truncatedDiff := diff[:remaining] + "\n[diff truncated to fit context window]"
	return expanded + "\n\n" + truncatedDiff, true
}
