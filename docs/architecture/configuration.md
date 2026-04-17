# Module: Configuration (`internal/config/`)

## Responsibility

Load, merge, and validate the YAML configuration. Implements the three-tier resolution order: local → global → built-in defaults.

## Package Structure

```
internal/config/
  config.go       # Config struct, Load(), merge logic
  defaults.go     # Built-in default values
  config_test.go  # Unit tests
```

## Config Struct

```go
type Config struct {
    Provider       string            `yaml:"provider"`
    ModelID        string            `yaml:"model_id"`
    Prompt         string            `yaml:"prompt"`
    ProviderConfig map[string]string `yaml:"provider_config"`
}
```

## Resolution Order

When no `--config` override is provided:

1. `.scry/config.yaml` (current working directory)
2. `~/.scry/config.yaml` (user home)
3. Built-in defaults hardcoded in `defaults.go`

When `--config <path>` is provided, only that file and built-in defaults are used (local and global are skipped).

For each field, the first non-zero value found wins. This is a field-level merge, not file-level — a partial local config fills missing fields from global, then from defaults.

## Built-in Defaults

```go
var Defaults = Config{
    Provider: "bedrock",
    ModelID:  "global.amazon.nova-2-lite-v1:0",
    Prompt:   "<type>(<scope>): <short summary>...",  // conventional commit format instructions
    ProviderConfig: map[string]string{
        "region": "us-east-1",
    },
}
```

Provider-specific keys in `ProviderConfig`:

| Provider | Key | Default | Description |
|---|---|---|---|
| `bedrock` | `region` | `us-east-1` | AWS region for Bedrock API calls |
| `ollama` | `endpoint` | `http://localhost:11434` | Ollama server URL |

## Load Function

```go
// Load resolves configuration with three-tier precedence:
// override path (exclusive) → local → global → defaults.
// cwd and homeDir are injected for testability.
func Load(overridePath, cwd, homeDir string) (*Config, error)
```

## Validation

- YAML parse errors include the file path in the error message. `yaml.v3` provides parse details via `yaml.TypeError`.
- Unknown fields are ignored (forward compatibility).

## Error Types

- `ConfigParseError{Path, Message}` — wraps YAML parse failures with the file path.

## Dependencies

- `gopkg.in/yaml.v3`
- `os` (file reading, home dir)

## Relevant Requirements

REQ-U-003, REQ-U-006, REQ-X-006
