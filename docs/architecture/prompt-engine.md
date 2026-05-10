# Module: Prompt Engine (`internal/prompt/`)

## Responsibility

Expand template variables in the configured prompt, assemble the final payload (prompt + diff), truncate the diff if it exceeds the model's context window, and build per-file summary prompts for large diffs.

## Package Structure

```
internal/prompt/
  prompt.go       # Build(), BuildFromSummaries(), SummaryPrompt(), expandVars(), estimateTokens()
  prompt_test.go  # Unit tests
```

## Public API

```go
// Build expands template variables in the prompt, appends the diff, and
// truncates if the total exceeds maxTokens. Returns the assembled string
// and a boolean indicating whether truncation occurred.
func Build(promptTemplate string, diff string, vars Vars, maxTokens int) (payload string, truncated bool)

// BuildFromSummaries assembles the final commit-generation prompt using
// per-file summaries instead of the raw diff.
func BuildFromSummaries(promptTemplate string, summaries map[string]string, vars Vars, maxTokens int) (string, bool)

// SummaryPrompt returns a prompt asking the LLM to summarize a single file's diff.
func SummaryPrompt(file, diff string) string

// Vars holds the values for template expansion.
type Vars struct {
    BranchName string
    Author     string
}
```

## Template Expansion

Simple string replacement via `strings.NewReplacer` — no need for `text/template`:
- `{{branch_name}}` → `vars.BranchName`
- `{{author}}` → `vars.Author`

Unknown `{{...}}` tokens are left as-is (no error).

## Truncation Strategy

- A `maxTokens` of 0 means unlimited (no truncation).
- Estimate token count as `len(text) / 4` (conservative approximation for English text).
- If prompt + diff exceeds `maxTokens`, truncate the **diff** from the end to fit.
- The prompt is never truncated.
- When truncation occurs, append a marker: `\n[diff truncated to fit context window]`
- Return `truncated = true` so the CLI can warn the user.

## Per-File Summary Mode

When the diff exceeds the configured `diff_summary_threshold`, the CLI uses the summary workflow:

1. `SummaryPrompt(file, diff)` generates a prompt asking the LLM to summarize a single file's changes in 2-3 sentences.
2. The CLI invokes the LLM for each file concurrently.
3. `BuildFromSummaries(promptTemplate, summaries, vars, maxTokens)` assembles the final prompt by:
   - Expanding template variables
   - Formatting per-file summaries as a structured list
   - Delegating to `Build()` for final truncation if needed

## Context Window Limits

The `maxTokens` value is passed in by the caller (CLI layer), sourced from a hardcoded map of known model context sizes in the provider package. Default fallback: 128,000 tokens.

## Dependencies

- `strings`
