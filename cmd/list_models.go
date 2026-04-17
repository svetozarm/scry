package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/svetozarm/scry/internal/config"
	"github.com/svetozarm/scry/internal/provider"
	"github.com/svetozarm/scry/internal/ui"
)

var listModelsCmd = &cobra.Command{
	Use:   "list-models",
	Short: "List available models from the configured provider",
	RunE:  runListModels,
}

func runListModels(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return handleError(err)
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return handleError(err)
	}

	cfg, err := config.Load(configPath, cwd, homeDir)
	if err != nil {
		return handleError(err)
	}

	p, err := provider.New(cfg.Provider, cfg.ProviderConfig)
	if err != nil {
		return handleError(err)
	}

	models, err := p.ListModels(cmd.Context())
	if err != nil {
		return handleError(err)
	}

	if nonInteractive {
		for _, m := range models {
			fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", m.ID, m.Name)
		}
	} else {
		ui.DisplayModels(models)
	}
	return nil
}
