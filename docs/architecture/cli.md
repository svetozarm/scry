# Module: CLI (`cmd/`)

## Responsibility

Entry point and orchestrator. Defines the Cobra command tree, parses flags, and wires together all internal modules to execute the two workflows: **generate** (default) and **list-models**.

## Package Structure

```
cmd/
  root.go          # Root command, global flags, config loading
  generate.go      # Default command — the generate workflow
  list_models.go   # `list-models` subcommand
  errors.go        # Error mapping, sanitization, exit codes
```

## Commands & Flags

| Command | Description |
|---|---|
| `scry` (root/default) | Generate a commit message from staged changes |
| `scry list-models` | List available models from the configured provider |

| Flag | Scope | Type | Default | Description |
|---|---|---|---|---|
| `--non-interactive` | root | bool | false | Output plain text to stdout, no prompts/styling/spinner |
| `--config` | root | string | "" | Override config file path |

## Orchestration — Generate

The interactive loop uses a `Prompter` interface (defined in `internal/ui/`) for testability. The default implementation delegates to the real UI functions.

```
func runGenerate(cmd, args):
    cwd := os.Getwd()
    homeDir := os.UserHomeDir()
    cfg := config.Load(configPath, cwd, homeDir)
    git.EnsureRepo(cwd)
    git.EnsureStagedChanges(cwd)
    diff := git.Diff(cwd)
    branch := git.BranchName(cwd)
    author := git.Author(cwd)
    provider := provider.New(cfg.Provider, cfg.ProviderConfig)
    payload, truncated := prompt.Build(cfg.Prompt, diff, vars, provider.MaxTokens(cfg.ModelID))

    if truncated:
        warn about truncation

    if nonInteractive:
        msg := provider.Invoke(ctx, cfg.ModelID, payload)
        fmt.Fprintln(stdout, msg)
        return

    loop:
        msg := prompter.WithSpinner("Generating...", func() { return provider.Invoke(ctx, cfg.ModelID, payload) })
        prompter.DisplayMessage(msg)
        choice := prompter.PromptAction()  // accept | regenerate | cancel
        switch choice:
            accept → git.Commit(cwd, msg); prompter.DisplayCommitResult()
            regenerate → continue loop
            cancel → exit 0
```

## Error Handling

All errors bubble up to the CLI layer. The `handleError` function maps error types to user-friendly messages and exit codes:

| Error | Exit Code |
|---|---|
| Git errors (no repo, no staged changes, commit failed) | 2 |
| Provider errors (auth, rate limit, timeout, model not found) | 3 |
| Config parse errors | 4 |
| Unknown errors | 1 |

In interactive mode, errors are rendered via `ui.DisplayError()`. In non-interactive mode, plain text is written to stderr. Errors containing credentials are sanitized before display.

Errors are wrapped in a `silentError` type so Cobra doesn't print them again (since `SilenceErrors: true`).

## Dependencies

- `internal/config`
- `internal/git`
- `internal/prompt`
- `internal/provider`
- `internal/ui`

## Relevant Requirements

REQ-E-001, REQ-E-002, REQ-E-003, REQ-E-004, REQ-E-005, REQ-E-006, REQ-E-007, REQ-NF-006
