# Module: CLI (`cmd/`)

## Responsibility

Entry point and orchestrator. Defines the Cobra command tree, parses flags, and wires together all internal modules to execute the three workflows: **generate** (default), **list-models**, and **update**.

## Package Structure

```
cmd/
  root.go          # Root command, global flags, config loading
  generate.go      # Default command — the generate workflow
  list_models.go   # `list-models` subcommand
  update.go        # `update` subcommand — self-update workflow
  errors.go        # Error mapping, sanitization, exit codes
```

## Commands & Flags

| Command | Description |
|---|---|
| `scry` (root/default) | Generate a commit message from staged changes |
| `scry list-models` | List available models from the configured provider |
| `scry update` | Self-update to the latest GitHub release |

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
    maxTokens := provider.MaxTokens(cfg.ModelID)

    // Large diff summary mode
    if cfg.DiffSummaryThreshold > 0 && len(diff) > cfg.DiffSummaryThreshold:
        summaries := summarizeFiles(ctx, cwd, provider, cfg, nonInteractive)
        payload, truncated = prompt.BuildFromSummaries(cfg.Prompt, summaries, vars, maxTokens)
    else:
        payload, truncated = prompt.Build(cfg.Prompt, diff, vars, maxTokens)

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

## Orchestration — Update

```
func runUpdate(cmd, args):
    checker := update.NewGitHubClient(repo)
    result := update.Run(ctx, Version, checker)
    if ErrDevBuild → display warning, exit 0
    if result.UpToDate → display "already up to date"
    else → display "Updated: old → new"
```

## Per-File Summary Worker Pool

When the diff exceeds `cfg.DiffSummaryThreshold`, `summarizeFiles` spawns a worker pool of `cfg.SummaryConcurrency` goroutines. Each worker:
1. Gets the per-file diff via `git.DiffFile`
2. Builds a summary prompt via `prompt.SummaryPrompt`
3. Invokes the LLM provider
4. Reports progress via `ui.DisplayProgressBar` (or stderr in non-interactive mode)

## Error Handling

All errors bubble up to the CLI layer. The `handleError` function maps error types to user-friendly messages and exit codes:

| Error | Exit Code |
|---|---|
| Git errors (no repo, no staged changes, commit failed) | 2 |
| Provider errors (auth, rate limit, timeout, model not found) | 3 |
| Config parse errors | 4 |
| Update errors (API, checksum, signature, asset not found, permission, replace) | 5 |
| Unknown errors | 1 |

In interactive mode, errors are rendered via `ui.DisplayError()`. In non-interactive mode, plain text is written to stderr. Errors containing credentials are sanitized before display.

Errors are wrapped in a `silentError` type so Cobra doesn't print them again (since `SilenceErrors: true`).

## Dependencies

- `internal/config`
- `internal/git`
- `internal/prompt`
- `internal/provider`
- `internal/ui`
- `internal/update`
