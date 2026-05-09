package prompt

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExpandVars_BranchName(t *testing.T) {
	got := expandVars("on {{branch_name}}", Vars{BranchName: "main"})
	assert.Equal(t, "on main", got)
}

func TestExpandVars_Author(t *testing.T) {
	got := expandVars("by {{author}}", Vars{Author: "Jane"})
	assert.Equal(t, "by Jane", got)
}

func TestExpandVars_Both(t *testing.T) {
	got := expandVars("{{branch_name}} by {{author}}", Vars{BranchName: "feat/x", Author: "Jane"})
	assert.Equal(t, "feat/x by Jane", got)
}

func TestExpandVars_UnknownToken(t *testing.T) {
	got := expandVars("hello {{unknown}}", Vars{BranchName: "main"})
	assert.Equal(t, "hello {{unknown}}", got)
}

func TestBuild_PromptPrependedToDiff(t *testing.T) {
	payload, truncated := Build("Generate a commit message", "diff --git a/file", Vars{}, 0)
	assert.Equal(t, "Generate a commit message\n\ndiff --git a/file", payload)
	assert.False(t, truncated)
}

func TestBuild_NoTruncation(t *testing.T) {
	prompt := "prompt"
	diff := "small diff"
	payload, truncated := Build(prompt, diff, Vars{}, 1000)
	assert.Contains(t, payload, diff)
	assert.False(t, truncated)
}

func TestBuild_Truncation(t *testing.T) {
	prompt := "prompt"
	diff := strings.Repeat("x", 1000)
	// prompt="prompt" is 6 chars → 6/4=1 token, separator "\n\n" is 2 chars → 0 tokens
	// Allow 10 tokens total → 40 chars total budget
	payload, truncated := Build(prompt, diff, Vars{}, 10)
	assert.True(t, truncated)
	assert.Contains(t, payload, "[diff truncated to fit context window]")
	assert.Less(t, len(payload), len(prompt)+len(diff))
}

func TestBuild_PromptExceedsLimit(t *testing.T) {
	prompt := strings.Repeat("x", 100)
	diff := "some diff"
	// prompt is 100 chars → 25 tokens, limit is 5
	payload, truncated := Build(prompt, diff, Vars{}, 5)
	assert.True(t, truncated)
	assert.Equal(t, prompt, payload)
}

func TestBuild_MaxTokensZero_Unlimited(t *testing.T) {
	prompt := "prompt"
	diff := strings.Repeat("x", 10000)
	payload, truncated := Build(prompt, diff, Vars{}, 0)
	assert.False(t, truncated)
	assert.Equal(t, prompt+"\n\n"+diff, payload)
}

func TestBuild_ExpandsVars(t *testing.T) {
	payload, _ := Build("on {{branch_name}} by {{author}}", "diff", Vars{BranchName: "main", Author: "Jane"}, 0)
	assert.Equal(t, "on main by Jane\n\ndiff", payload)
}

func TestSummaryPrompt_ContainsFileAndDiff(t *testing.T) {
	p := SummaryPrompt("main.go", "+func main() {}")
	assert.Contains(t, p, "main.go")
	assert.Contains(t, p, "+func main() {}")
}

func TestBuildFromSummaries_AssemblesPayload(t *testing.T) {
	summaries := map[string]string{
		"a.go": "Added new handler",
	}
	payload, truncated := BuildFromSummaries("Generate commit", summaries, Vars{BranchName: "main"}, 0)
	assert.False(t, truncated)
	assert.Contains(t, payload, "Generate commit")
	assert.Contains(t, payload, "## a.go")
	assert.Contains(t, payload, "Added new handler")
}

func TestBuildFromSummaries_ExpandsVars(t *testing.T) {
	summaries := map[string]string{"f.go": "change"}
	payload, _ := BuildFromSummaries("on {{branch_name}} by {{author}}", summaries, Vars{BranchName: "dev", Author: "Jo"}, 0)
	assert.Contains(t, payload, "on dev by Jo")
}
