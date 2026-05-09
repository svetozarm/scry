package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestConfigYAMLRoundTrip(t *testing.T) {
	input := `
provider: bedrock
model_id: amazon.nova-2-lite-v1:0
prompt: Generate a commit message
provider_config:
  region: us-east-1
`
	var cfg Config
	require.NoError(t, yaml.Unmarshal([]byte(input), &cfg))

	assert.Equal(t, "bedrock", cfg.Provider)
	assert.Equal(t, "amazon.nova-2-lite-v1:0", cfg.ModelID)
	assert.Equal(t, "Generate a commit message", cfg.Prompt)
	assert.Equal(t, map[string]string{"region": "us-east-1"}, cfg.ProviderConfig)
}

func TestConfigYAMLPartial(t *testing.T) {
	input := `model_id: custom-model`
	var cfg Config
	require.NoError(t, yaml.Unmarshal([]byte(input), &cfg))

	assert.Equal(t, "custom-model", cfg.ModelID)
	assert.Empty(t, cfg.Provider)
	assert.Empty(t, cfg.Prompt)
	assert.Nil(t, cfg.ProviderConfig)
}

func TestDefaults(t *testing.T) {
	assert.Equal(t, "bedrock", Defaults.Provider)
	assert.Equal(t, "openai.gpt-oss-20b-1:0", Defaults.ModelID)
	assert.NotEmpty(t, Defaults.Prompt)
	assert.Contains(t, Defaults.Prompt, "{{branch_name}}")
	assert.Contains(t, Defaults.Prompt, "{{author}}")
	assert.Equal(t, "us-east-1", Defaults.ProviderConfig["region"])
}

// writeYAML is a test helper that writes a YAML config file at the given path.
func writeYAML(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func TestMergeOverlayOverridesBase(t *testing.T) {
	base := Config{
		Provider: "bedrock",
		ModelID:  "base-model",
		Prompt:   "base prompt",
		ProviderConfig: map[string]string{
			"region": "us-east-1",
			"key":    "base-key",
		},
	}
	overlay := Config{
		Provider: "openai",
		ModelID:  "overlay-model",
		Prompt:   "overlay prompt",
		ProviderConfig: map[string]string{
			"region": "eu-west-1",
			"extra":  "new-val",
		},
	}
	result := merge(base, overlay)
	assert.Equal(t, "openai", result.Provider)
	assert.Equal(t, "overlay-model", result.ModelID)
	assert.Equal(t, "overlay prompt", result.Prompt)
	assert.Equal(t, "eu-west-1", result.ProviderConfig["region"])
	assert.Equal(t, "new-val", result.ProviderConfig["extra"])
	assert.Equal(t, "base-key", result.ProviderConfig["key"])
}

func TestMergeZeroOverlayKeepsBase(t *testing.T) {
	base := Config{
		Provider: "bedrock",
		ModelID:  "base-model",
		Prompt:   "base prompt",
		ProviderConfig: map[string]string{
			"region": "us-east-1",
		},
	}
	overlay := Config{} // all zero values
	result := merge(base, overlay)
	assert.Equal(t, "bedrock", result.Provider)
	assert.Equal(t, "base-model", result.ModelID)
	assert.Equal(t, "base prompt", result.Prompt)
	assert.Equal(t, "us-east-1", result.ProviderConfig["region"])
}

func TestMergePartialOverlay(t *testing.T) {
	base := Config{
		Provider: "bedrock",
		ModelID:  "base-model",
		Prompt:   "base prompt",
		ProviderConfig: map[string]string{
			"region": "us-east-1",
		},
	}
	overlay := Config{ModelID: "custom-model"}
	result := merge(base, overlay)
	assert.Equal(t, "bedrock", result.Provider)
	assert.Equal(t, "custom-model", result.ModelID)
	assert.Equal(t, "base prompt", result.Prompt)
	assert.Equal(t, "us-east-1", result.ProviderConfig["region"])
}

func TestMergeNilProviderConfigPreservesBase(t *testing.T) {
	base := Config{
		ProviderConfig: map[string]string{"region": "us-east-1"},
	}
	overlay := Config{ProviderConfig: nil}
	result := merge(base, overlay)
	assert.Equal(t, "us-east-1", result.ProviderConfig["region"])
}

func TestMergeThreeTierChain(t *testing.T) {
	defaults := Config{
		Provider: "bedrock",
		ModelID:  "default-model",
		Prompt:   "default prompt",
		ProviderConfig: map[string]string{
			"region": "us-east-1",
		},
	}
	global := Config{ModelID: "global-model"}
	local := Config{
		ProviderConfig: map[string]string{"region": "ap-southeast-1"},
	}
	result := merge(merge(defaults, global), local)
	assert.Equal(t, "bedrock", result.Provider)
	assert.Equal(t, "global-model", result.ModelID)
	assert.Equal(t, "default prompt", result.Prompt)
	assert.Equal(t, "ap-southeast-1", result.ProviderConfig["region"])
}

func TestLoadDefaultsOnly(t *testing.T) {
	// No config files exist → returns defaults.
	tmp := t.TempDir()
	cfg, err := Load("", tmp, filepath.Join(tmp, "fakehome"))
	require.NoError(t, err)
	assert.Equal(t, Defaults.Provider, cfg.Provider)
	assert.Equal(t, Defaults.ModelID, cfg.ModelID)
	assert.Equal(t, Defaults.Prompt, cfg.Prompt)
	assert.Equal(t, Defaults.ProviderConfig["region"], cfg.ProviderConfig["region"])
}

func TestLoadGlobalFallback(t *testing.T) {
	// No local file, global sets model_id → merged with defaults.
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	globalFile := filepath.Join(home, ".scry", "config.yaml")
	writeYAML(t, globalFile, `model_id: global-model`)

	cfg, err := Load("", tmp, home)
	require.NoError(t, err)
	assert.Equal(t, "global-model", cfg.ModelID)
	assert.Equal(t, Defaults.Provider, cfg.Provider) // from defaults
}

func TestLoadLocalPrecedence(t *testing.T) {
	// Both local and global exist → local wins.
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	localFile := filepath.Join(tmp, ".scry", "config.yaml")
	globalFile := filepath.Join(home, ".scry", "config.yaml")
	writeYAML(t, localFile, `model_id: local-model`)
	writeYAML(t, globalFile, `model_id: global-model`)

	cfg, err := Load("", tmp, home)
	require.NoError(t, err)
	assert.Equal(t, "local-model", cfg.ModelID)
}

func TestLoadOverridePath(t *testing.T) {
	// Override path is used exclusively.
	tmp := t.TempDir()
	overrideFile := filepath.Join(tmp, "custom.yaml")
	writeYAML(t, overrideFile, `model_id: override-model`)

	cfg, err := Load(overrideFile, tmp, tmp)
	require.NoError(t, err)
	assert.Equal(t, "override-model", cfg.ModelID)
	assert.Equal(t, Defaults.Provider, cfg.Provider)
}

func TestLoadOverridePathNotFound(t *testing.T) {
	// Override path doesn't exist → error.
	tmp := t.TempDir()
	_, err := Load(filepath.Join(tmp, "missing.yaml"), tmp, tmp)
	require.Error(t, err)
}

func TestLoadInvalidYAML(t *testing.T) {
	// Invalid YAML → ConfigParseError with file path and line number.
	tmp := t.TempDir()
	badFile := filepath.Join(tmp, ".scry", "config.yaml")
	writeYAML(t, badFile, `{{{`)

	_, err := Load("", tmp, filepath.Join(tmp, "fakehome"))
	require.Error(t, err)
	var parseErr *ConfigParseError
	require.ErrorAs(t, err, &parseErr)
	assert.Equal(t, badFile, parseErr.Path)
	assert.Contains(t, parseErr.Message, "line")
	assert.Contains(t, parseErr.Error(), "invalid config")
	assert.Contains(t, parseErr.Error(), badFile)
}

func TestLoadInvalidYAMLWithLineNumber(t *testing.T) {
	// Error on line 2 reports correct line number.
	tmp := t.TempDir()
	badFile := filepath.Join(tmp, ".scry", "config.yaml")
	writeYAML(t, badFile, "provider: bedrock\nmodel_id: {{{")

	_, err := Load("", tmp, filepath.Join(tmp, "fakehome"))
	require.Error(t, err)
	var parseErr *ConfigParseError
	require.ErrorAs(t, err, &parseErr)
	assert.Contains(t, parseErr.Message, "line 2")
}

func TestLoadProviderConfigMerge(t *testing.T) {
	// Local overrides one key, default fills the rest.
	tmp := t.TempDir()
	localFile := filepath.Join(tmp, ".scry", "config.yaml")
	writeYAML(t, localFile, "provider_config:\n  region: eu-west-1\n  api_key: secret")

	cfg, err := Load("", tmp, filepath.Join(tmp, "fakehome"))
	require.NoError(t, err)
	assert.Equal(t, "eu-west-1", cfg.ProviderConfig["region"])
	assert.Equal(t, "secret", cfg.ProviderConfig["api_key"])
}
