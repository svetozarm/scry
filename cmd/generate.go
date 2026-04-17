package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/svetozarm/scry/internal/config"
	"github.com/svetozarm/scry/internal/git"
	"github.com/svetozarm/scry/internal/prompt"
	"github.com/svetozarm/scry/internal/provider"
	"github.com/svetozarm/scry/internal/ui"
)

// prompter is the UI abstraction used by the interactive loop.
// Override in tests with a mock.
var prompter ui.Prompter = ui.DefaultPrompter{}

func runGenerate(cmd *cobra.Command, args []string) error {
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

	if err := git.EnsureRepo(cwd); err != nil {
		return handleError(err)
	}
	if err := git.EnsureStagedChanges(cwd); err != nil {
		return handleError(err)
	}

	diff, err := git.Diff(cwd)
	if err != nil {
		return handleError(err)
	}
	branch, _ := git.BranchName(cwd)
	author, _ := git.Author(cwd)

	p, err := provider.New(cfg.Provider, cfg.ProviderConfig)
	if err != nil {
		return handleError(err)
	}

	vars := prompt.Vars{BranchName: branch, Author: author}
	maxTokens := p.MaxTokens(cfg.ModelID)
	payload, truncated := prompt.Build(cfg.Prompt, diff, vars, maxTokens)
	if truncated {
		if nonInteractive {
			fmt.Fprintln(os.Stderr, "Warning: diff was truncated to fit context window")
		} else {
			ui.DisplayWarning("diff was truncated to fit context window")
		}
	}

	if nonInteractive {
		msg, err := p.Invoke(cmd.Context(), cfg.ModelID, payload)
		if err != nil {
			return handleError(err)
		}
		fmt.Fprintln(cmd.OutOrStdout(), msg)
		return nil
	}

	for {
		msg, err := prompter.WithSpinner("Generating commit message...", func() (string, error) {
			return p.Invoke(cmd.Context(), cfg.ModelID, payload)
		})
		if err != nil {
			return handleError(err)
		}

		prompter.DisplayMessage(msg)

		action, err := prompter.PromptAction()
		if err != nil {
			return handleError(err)
		}

		switch action {
		case ui.ActionAccept:
			output, err := git.Commit(cwd, msg)
			if err != nil {
				return handleError(err)
			}
			prompter.DisplayCommitResult(output)
			return nil
		case ui.ActionRegenerate:
			continue
		case ui.ActionCancel:
			return nil
		}
	}
}
