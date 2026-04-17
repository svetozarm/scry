package cmd

import (
	"github.com/spf13/cobra"

	_ "github.com/svetozarm/scry/internal/provider/bedrock"
	_ "github.com/svetozarm/scry/internal/provider/ollama"
)

var (
	nonInteractive bool
	configPath     string
)

var rootCmd = &cobra.Command{
	Use:           "scry",
	Short:         "Generate commit messages from staged changes using an LLM",
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE:          runGenerate,
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&nonInteractive, "non-interactive", false, "Output plain text, no prompts or styling")
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "Path to config file")
	rootCmd.AddCommand(listModelsCmd)
}

func Execute() error {
	return rootCmd.Execute()
}
