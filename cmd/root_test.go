package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootCommandUseName(t *testing.T) {
	assert.Equal(t, "scry", rootCmd.Use)
}

func TestRootCommandHasNonInteractiveFlag(t *testing.T) {
	f := rootCmd.PersistentFlags().Lookup("non-interactive")
	require.NotNil(t, f, "--non-interactive flag should be registered")
	assert.Equal(t, "false", f.DefValue)
}

func TestRootCommandHasConfigFlag(t *testing.T) {
	f := rootCmd.PersistentFlags().Lookup("config")
	require.NotNil(t, f, "--config flag should be registered")
	assert.Equal(t, "", f.DefValue)
}

func TestRootCommandSilencesErrors(t *testing.T) {
	assert.True(t, rootCmd.SilenceErrors, "SilenceErrors should be true")
}

func TestRootCommandSilencesUsage(t *testing.T) {
	assert.True(t, rootCmd.SilenceUsage, "SilenceUsage should be true")
}

func TestListModelsSubcommandRegistered(t *testing.T) {
	found := false
	for _, c := range rootCmd.Commands() {
		if c.Use == "list-models" {
			found = true
			break
		}
	}
	assert.True(t, found, "list-models subcommand should be registered")
}
