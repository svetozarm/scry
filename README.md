# Scry

A single-binary CLI that generates commit messages from your staged changes using an LLM. Stage your changes, run `scry`, and get a well-crafted conventional commit message — accept it, regenerate, or cancel.

![Scry demo](https://github.com/user-attachments/assets/1f38a405-0bca-45c2-8e1d-7ed3eae16bab)

## Installation

### Go Install

```bash
go install github.com/svetozarm/scry@latest
```

### Download Binary

Download a prebuilt binary from the [Releases](https://github.com/svetozarm/scry/releases) page. Binaries are available for Linux, macOS, and Windows (amd64 and arm64).

## Quick Start

```bash
# Stage your changes
git add .

# Generate a commit message
scry
```

Scry will show a spinner while the LLM generates a message, then display it and prompt you to **accept**, **regenerate**, or **cancel**.

## Configuration

Scry loads configuration from YAML files with three-tier precedence (plus an explicit override):

1. `--config <path>` (explicit override — bypasses local and global)
2. `.scry/config.yaml` (local, per-project)
3. `~/.scry/config.yaml` (global)
4. Built-in defaults

Fields are merged individually: a partial local config fills missing fields from global, then from defaults.

```yaml
# Provider to use ("bedrock" or "ollama")
provider: bedrock

# Model ID to invoke
model_id: global.amazon.nova-2-lite-v1:0

# Prompt template (supports {{branch_name}} and {{author}} variables)
prompt: |
  <type>(<scope>): <short summary>

  <optional body>

  Types: feat, fix, docs, style, refactor, test, chore, perf, ci, build
  Rules:

  Subject line — imperative mood ("add" not "added"), lowercase, no period, ≤50 chars
  Scope — optional, names the affected module (e.g., feat(auth): add OAuth flow)
  Body — wrap at 72 chars, explain what and why, not how

  Generate a concise, modern commit message for the
  following staged changes on branch {{branch_name}} by {{author}}.
  Use imperative mood.
  IMPORTANT: Output only the commit message, nothing else. No code blocks, no markdown notation. Just plaintext commit message

# Provider-specific settings
provider_config:
  region: us-east-1
```

## Commands

### `scry` (default)

Generates a commit message from staged changes. Runs the interactive loop: generate → display → accept/regenerate/cancel.

### `scry list-models`

Lists available models from the configured provider.

## Flags

| Flag | Description |
|---|---|
| `--non-interactive` | Output plain text to stdout with no styling, spinner, or prompts. Does not commit. |
| `--config <path>` | Path to a specific config file (overrides local and global). |

## Provider Setup

### Amazon Bedrock

Scry uses Amazon Bedrock as its LLM provider. You need valid AWS credentials configured via any method supported by the AWS SDK:

- Environment variables (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`)
- Shared credentials file (`~/.aws/credentials`)
- IAM role (EC2, ECS, Lambda)
- SSO (`aws sso login`)

Ensure the configured IAM identity has permissions for `bedrock:Converse` and `bedrock:ListFoundationModels`.

Set the region in your config:

```yaml
provider_config:
  region: us-east-1
```

### Ollama

Scry supports [Ollama](https://ollama.com/) as a local LLM provider. Install Ollama and pull a model, then configure Scry to use it:

```yaml
provider: ollama
model_id: llama3
provider_config:
  endpoint: http://localhost:11434
```

The `endpoint` defaults to `http://localhost:11434` and can be omitted if Ollama is running on the default address.

## Cost

Scry calls an LLM inference API on each run. Depending on the provider and model used, this may incur costs. Check your provider's pricing page for details.

## License

MIT
