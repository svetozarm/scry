# Module: Prompt Engine (`internal/prompt/`)

## Responsibility

Expand template variables in the configured prompt, assemble the final payload (prompt + diff), and truncate the diff if it exceeds the model's context window.

## Package Structure

```
internal/prompt/
  prompt.go       # Build(), expandVars(), estimateTokens()
  prompt_test.go  # Unit tests
```

## Public API

```go
// Build expands template variables in the prompt, appends the diff, and
// truncates if the total exceeds maxTokens. Returns the assembled string
// and a boolean indicating whether truncation occurred.
func Build(promptTemplate string, diff string, vars Vars, maxTokens int) (payload string, truncated bool)

// Vars holds the values for template expansion.
type Vars struct {
    BranchName string
    Author     string
}
```

## Template Expansion

Simple string replacement — no need for `text/template`:
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

## Context Window Limits

The `maxTokens` value is passed in by the caller (CLI layer), sourced from a hardcoded map of known model context sizes in the provider package. Default fallback: 128,000 tokens.

## Dependencies

- `strings`

## Relevant Requirements

REQ-U-005, REQ-X-005
