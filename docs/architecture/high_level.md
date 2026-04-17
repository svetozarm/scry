# Scry — High-Level Architecture

## Overview

Scry is a single-binary Go CLI that generates commit messages by piping `git diff --cached` output through an LLM provider. It follows a linear pipeline architecture with pluggable provider backends.

## System Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                        CLI Entry (Cobra)                        │
│  Parses commands & flags: generate (default), list-models       │
│  Flags: --non-interactive                                       │
└──────────────┬──────────────────────────────────────────────────┘
               │
               ▼
┌──────────────────────────┐
│     Configuration        │
│  .scry/config.yaml│
│  ~/.scry/config.yaml│
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
               ▼
┌──────────────────────────┐
│   Bedrock Provider       │
│  AWS SDK Go v2           │
│  Converse API            │
│  ListFoundationModels    │
│  Reads provider_config   │
└──────────────────────────┘
               │
               ▼
┌──────────────────────────────────────────────────────────────┐
│                     Terminal UI (Charm)                        │
│  Lip Gloss: styled output (commit message, errors, warnings) │
│  Huh: inline prompts (accept / regenerate / cancel)           │
│  Spinner: inline loading indicator                            │
└───────────────────────────────────────────────────────────────┘
```

## Modules

| Module | Package | Responsibility |
|---|---|---|
| **CLI** | `cmd/` | Cobra command tree, flag parsing, orchestration of the generate and list-models workflows |
| **Configuration** | `internal/config/` | YAML loading, merge logic (local → global → defaults), validation |
| **Git Integration** | `internal/git/` | Repo detection, staged change detection, diff retrieval, commit execution |
| **Prompt Engine** | `internal/prompt/` | Template variable expansion, prompt + diff assembly, context window truncation |
| **Provider Abstraction** | `internal/provider/` | `Provider` interface, error type classification, provider registry/factory |
| **Bedrock Provider** | `internal/provider/bedrock/` | AWS SDK v2 implementation of the Provider interface |
| **Terminal UI** | `internal/ui/` | Styled output, spinner, interactive prompts (accept/regenerate/cancel), non-interactive plain output |

## Data Flow — Generate Command

1. **CLI** parses flags, loads **Configuration**
2. **Git Integration** validates repo & staged changes, retrieves diff
3. **Prompt Engine** expands template variables, prepends prompt to diff, truncates if needed
4. **Provider Abstraction** routes to **Bedrock Provider**, which calls Converse
5. **Terminal UI** shows spinner during inference, then displays result
6. **Terminal UI** prompts user: accept / regenerate / cancel
7. On accept → **Git Integration** runs `git commit -m`
8. On regenerate → loop back to step 4
9. On cancel → exit 0

## Data Flow — List Models Command

1. **CLI** parses flags, loads **Configuration**
2. **Provider Abstraction** routes to **Bedrock Provider**, which calls ListFoundationModels
3. **Terminal UI** displays model list

## Non-Interactive Mode

Steps 1–4 are identical. Step 5 writes plain text to stdout with no styling, no spinner, no prompt. Exit 0.

## Error Handling Strategy

All provider errors are classified into typed errors at the abstraction layer:
- `ErrAuth` → REQ-X-001
- `ErrRateLimit` → REQ-X-002
- `ErrTimeout` → REQ-X-003
- `ErrModelNotFound` → REQ-X-004

The CLI layer maps these to styled error messages and non-zero exit codes. Git and config errors are handled similarly with specific exit codes.

## Cross-Cutting Concerns

- **No disk persistence** of diffs, messages, or credentials (REQ-NF-004)
- **No outbound traffic** except to the configured LLM provider (REQ-NF-003)
- **Cross-platform** via standard Go cross-compilation (REQ-NF-005)
- **Memory budget** ≤ 50 MB resident (REQ-NF-002)
- **Startup latency** < 200 ms to first output (REQ-NF-001)
