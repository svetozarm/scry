package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/svetozarm/scry/internal/ui"
	"github.com/svetozarm/scry/internal/update"
)

const repo = "svetozarm/scry"

// newChecker creates a ReleaseChecker; overridable for testing.
var newChecker = func() update.ReleaseChecker {
	return update.NewGitHubClient(repo)
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update scry to the latest version",
	RunE:  runUpdate,
}

func init() {
	rootCmd.AddCommand(updateCmd)
}

func runUpdate(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	checker := newChecker()
	result, err := update.Run(ctx, Version, checker)
	if err != nil {
		if errors.Is(err, update.ErrDevBuild) {
			if nonInteractive {
				fmt.Fprintln(cmd.OutOrStdout(), "Warning: update unavailable for development builds")
			} else {
				ui.DisplayWarning("Update unavailable for development builds")
			}
			return nil
		}
		return handleError(err)
	}

	if result.UpToDate {
		msg := fmt.Sprintf("Already up to date: %s", Version)
		if nonInteractive {
			fmt.Fprintln(cmd.OutOrStdout(), msg)
		} else {
			fmt.Println(ui.SuccessStyle.Render("✓ " + msg))
		}
		return nil
	}

	msg := fmt.Sprintf("Updated: %s → %s", result.OldVersion, result.NewVersion)
	if nonInteractive {
		fmt.Fprintln(cmd.OutOrStdout(), msg)
	} else {
		fmt.Println(ui.SuccessStyle.Render("✓ " + msg))
	}
	return nil
}
