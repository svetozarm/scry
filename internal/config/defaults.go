package config

// Defaults contains the built-in default configuration values.
var Defaults = Config{
	Provider:             "bedrock",
	ModelID:              "global.amazon.nova-2-lite-v1:0",
	DiffSummaryThreshold: 32000,
	Prompt: `<type>(<scope>): <short summary>

<optional body>

Types: feat, fix, docs, style, refactor, test, chore, perf, ci, build
Rules:

Subject line — imperative mood ("add" not "added"), lowercase, no period, ≤50 chars
Scope — optional, names the affected module (e.g., feat(auth): add OAuth flow)
Body — wrap at 72 chars, explain what and why, not how

Generate a concise, modern commit message for the
following staged changes on branch {{branch_name}} by {{author}}.
Use imperative mood. 
IMPORTANT: Output only the commit message, nothing else. No code blocks, no markdown notation. Just plaintext commit message`,
	ProviderConfig: map[string]string{
		"region": "us-east-1",
	},
}
