package cmd

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateSubcommandRegistered(t *testing.T) {
	found := false
	for _, c := range rootCmd.Commands() {
		if c.Use == "update" {
			found = true
			break
		}
	}
	assert.True(t, found, "update subcommand should be registered")
}

func TestUpdateCommandShortDescription(t *testing.T) {
	var cmd *cobra.Command
	for _, c := range rootCmd.Commands() {
		if c.Use == "update" {
			cmd = c
			break
		}
	}
	require.NotNil(t, cmd)
	assert.Equal(t, "Update scry to the latest version", cmd.Short)
}
