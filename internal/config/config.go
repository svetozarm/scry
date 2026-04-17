package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds the application configuration loaded from YAML files.
type Config struct {
	Provider       string            `yaml:"provider"`
	ModelID        string            `yaml:"model_id"`
	Prompt         string            `yaml:"prompt"`
	ProviderConfig map[string]string `yaml:"provider_config"`
}

// ConfigParseError wraps YAML parse failures with the file path.
type ConfigParseError struct {
	Path    string
	Message string
}

func (e *ConfigParseError) Error() string {
	return fmt.Sprintf("invalid config %s: %s", e.Path, e.Message)
}

// merge returns a Config where overlay's non-zero fields override base.
// For ProviderConfig, keys are merged with overlay winning on conflicts.
func merge(base, overlay Config) Config {
	if overlay.Provider != "" {
		base.Provider = overlay.Provider
	}
	if overlay.ModelID != "" {
		base.ModelID = overlay.ModelID
	}
	if overlay.Prompt != "" {
		base.Prompt = overlay.Prompt
	}
	if overlay.ProviderConfig != nil {
		merged := make(map[string]string, len(base.ProviderConfig)+len(overlay.ProviderConfig))
		for k, v := range base.ProviderConfig {
			merged[k] = v
		}
		for k, v := range overlay.ProviderConfig {
			merged[k] = v
		}
		base.ProviderConfig = merged
	}
	return base
}

// loadFile reads and parses a YAML config file. Returns a zero Config and
// os.ErrNotExist if the file doesn't exist.
func loadFile(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Config{}, os.ErrNotExist
		}
		return Config{}, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, &ConfigParseError{Path: path, Message: err.Error()}
	}
	return cfg, nil
}

// Load resolves configuration with three-tier precedence:
// override path (exclusive) → local (.scry/config.yaml in cwd) → global (~/.scry/config.yaml) → defaults.
//
// cwd and homeDir are injected for testability.
func Load(overridePath, cwd, homeDir string) (*Config, error) {
	result := Defaults

	if overridePath != "" {
		cfg, err := loadFile(overridePath)
		if err != nil {
			return nil, err
		}
		result = merge(result, cfg)
		return &result, nil
	}

	// Global layer
	globalPath := filepath.Join(homeDir, ".scry", "config.yaml")
	globalCfg, err := loadFile(globalPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if err == nil {
		result = merge(result, globalCfg)
	}

	// Local layer (highest precedence)
	localPath := filepath.Join(cwd, ".scry", "config.yaml")
	localCfg, err := loadFile(localPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if err == nil {
		result = merge(result, localCfg)
	}

	return &result, nil
}
