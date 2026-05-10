# Scry — High-Level Architecture

## Overview

Scry is a single-binary Go CLI that generates commit messages by piping `git diff --cached` output through an LLM provider. It follows a linear pipeline architecture with pluggable provider backends. For large diffs, it can summarize per-file changes concurrently before generating the final commit message.

## System Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                        CLI Entry (Cobra)                        │
│  Parses commands & flags: generate (default), list-models,      │
│  update                                                         │
│  Flags: --non-interactive, --config                             │
└──────────────┬──────────────────────────────────────────────────┘
               │
               ▼
┌──────────────────────────┐
│     Configuration        │
│  .scry/config.yaml       │
│  ~/.scry/config.yaml     │
│  Built-in defaults       │
│  (YAML v3, merge logic)  │
└──────────┬───────────────┘
           │
           ▼
┌──────────────────────────┐     ┌──────────────────────────┐
│    Git Integration       │     │   Prompt Engine          │
│  Detect repo             │     │  Load template           │
│  Check staged changes    │     │  Expand {{branch_name}}, │
│  Run git diff --cached   │     │  {{author}}              │
│  Run git commit -m       │     │  Prepend to diff         │
│  Per-file diffs          │     │  Summary prompt builder  │
└──────────┬───────────────┘     └──────────┬───────────────┘
           │                                │
           └──────────┬─────────────────────┘
                      ▼
┌──────────────────────────────────────────────────────────────┐
│                  Provider Abstraction Layer                    │
│  Interface: Invoke(ctx, modelID, prompt) (string, error)      │
│             ListModels(ctx) ([]Model, error)                  │
│             MaxTokens(modelID) int                            │
│  Error classification: auth, rate-limit, timeout, model-404   │
└──────────────┬───────────────────────────────────────────────┘
               │
       ┌───────┴───────┐
       ▼               ▼
┌──────────────┐ ┌──────────────┐
│   Bedrock    │ │   Ollama     │
│   Provider   │ │   Provider   │
│  AWS SDK v2  │ │  REST API    │
│  Converse    │ │  /api/chat   │
└──────────────┘ └──────────────┘
               │
               ▼
┌──────────────────────────────────────────────────────────────┐
│                     Terminal UI (Charm)                        │
│  Lip Gloss: styled output (commit message, errors, warnings) │
│  Huh: inline prompts (accept / regenerate / cancel)           │
│  Spinner: inline loading indicator                            │
│  Progress bar: per-file summary progress                      │
└───────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────┐
│                     Self-Update Module                         │
│  GitHub Releases API, checksum verification, cosign signature │
│  Atomic binary replacement                                    │
└───────────────────────────────────────────────────────────────┘
```

## Modules

| Module | Package | Responsibility |
|---|---|---|
| **CLI** | `cmd/` | Cobra command tree, flag parsing, orchestration of the generate, list-models, and update workflows |
| **Configuration** | `internal/config/` | YAML loading, merge logic (local → global → defaults), validation |
| **Git Integration** | `internal/git/` | Repo detection, staged change detection, diff retrieval (full and per-file), commit execution |
| **Prompt Engine** | `internal/prompt/` | Template variable expansion, prompt + diff assembly, context window truncation, per-file summary prompts |
| **Provider Abstraction** | `internal/provider/` | `Provider` interface, error type classification, provider registry/factory |
| **Bedrock Provider** | `internal/provider/bedrock/` | AWS SDK v2 implementation of the Provider interface |
| **Ollama Provider** | `internal/provider/ollama/` | Ollama REST API implementation of the Provider interface |
| **Terminal UI** | `internal/ui/` | Styled output, spinner, progress bar, interactive prompts (accept/regenerate/cancel), non-interactive plain output |
| **Self-Update** | `internal/update/` | GitHub release checking, asset download, checksum/signature verification, atomic binary replacement |

## Data Flow — Generate Command

1. **CLI** parses flags, loads **Configuration**
2. **Git Integration** validates repo & staged changes, retrieves diff
3. If diff exceeds `diff_summary_threshold`, enter **summary mode**:
   a. **Git Integration** lists staged file names, retrieves per-file diffs
   b. **Prompt Engine** builds a summary prompt for each file
   c. **Provider** summarizes each file concurrently (worker pool of `summary_concurrency` goroutines)
   d. **Prompt Engine** assembles final prompt from per-file summaries
4. Otherwise, **Prompt Engine** expands template variables, prepends prompt to diff, truncates if needed
5. **Provider Abstraction** routes to configured provider, which calls the LLM
6. **Terminal UI** shows spinner during inference, then displays result
7. **Terminal UI** prompts user: accept / regenerate / cancel
8. On accept → **Git Integration** runs `git commit -m`
9. On regenerate → loop back to step 5
10. On cancel → exit 0

## Data Flow — List Models Command

1. **CLI** parses flags, loads **Configuration**
2. **Provider Abstraction** routes to configured provider, which lists available models
3. **Terminal UI** displays model list

## Data Flow — Update Command

1. **CLI** checks embedded version (exits with warning if dev build)
2. **Self-Update** queries GitHub Releases API for latest release
3. If already up to date, reports and exits
4. Downloads platform-specific archive, checksums file, and signature
5. Verifies cosign signature on checksums file
6. Verifies SHA-256 checksum of downloaded archive
7. Atomically replaces the current binary

## Non-Interactive Mode

Steps 1–5 are identical. Step 6 writes plain text to stdout with no styling, no spinner, no prompt. Exit 0.

## Error Handling Strategy

All provider errors are classified into typed errors at the abstraction layer:
- `ErrAuth` → authentication/authorisation failure
- `ErrRateLimit` → throttling / rate limit
- `ErrTimeout` → request timed out
- `ErrModelNotFound` → model ID not recognised

Update errors are classified separately:
- `ErrUpdateAPI` → GitHub API unreachable
- `ErrChecksumMismatch` → integrity check failed
- `ErrSignatureInvalid` → signature verification failed
- `ErrAssetNotFound` → no release asset for platform
- `ErrPermission` → binary path not writable
- `ErrReplaceFailed` → could not replace binary
- `ErrDevBuild` → no version embedded

The CLI layer maps these to styled error messages and non-zero exit codes. Git and config errors are handled similarly with specific exit codes. Errors containing credentials are sanitized before display.

## Cross-Cutting Concerns

- **No disk persistence** of diffs, messages, or credentials
- **No outbound traffic** except to the configured LLM provider and GitHub (for updates)
- **Cross-platform** via standard Go cross-compilation
- **Memory budget** ≤ 50 MB resident
- **Startup latency** < 200 ms to first output
