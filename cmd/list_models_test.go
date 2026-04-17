package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/svetozarm/scry/internal/provider"
)

func TestRunListModels_NonInteractive(t *testing.T) {
	mock := &mockProvider{
		models: []provider.Model{
			{ID: "model-1", Name: "Model One"},
			{ID: "model-2", Name: "Model Two"},
		},
	}
	registerMockProvider(mock)

	dir := t.TempDir()
	cfgDir := filepath.Join(dir, ".scry")
	require.NoError(t, os.MkdirAll(cfgDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(cfgDir, "config.yaml"), []byte("provider: mock\n"), 0644))

	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	defer os.Chdir(origDir)

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetArgs([]string{"list-models", "--non-interactive", "--config", filepath.Join(cfgDir, "config.yaml")})

	err := rootCmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "model-1\tModel One")
	assert.Contains(t, stdout.String(), "model-2\tModel Two")
}

func TestRunListModels_ProviderError(t *testing.T) {
	mock := &mockProvider{modelsErr: provider.ErrAuth}
	registerMockProvider(mock)

	dir := t.TempDir()
	cfgDir := filepath.Join(dir, ".scry")
	require.NoError(t, os.MkdirAll(cfgDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(cfgDir, "config.yaml"), []byte("provider: mock\n"), 0644))

	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	defer os.Chdir(origDir)

	rootCmd.SetArgs([]string{"list-models", "--non-interactive", "--config", filepath.Join(cfgDir, "config.yaml")})

	err := rootCmd.Execute()
	assert.Error(t, err)
}
