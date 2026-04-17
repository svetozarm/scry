package ui

import (
	"fmt"
	"os"

	"charm.land/lipgloss/v2"
	"github.com/svetozarm/scry/internal/provider"
)

// DisplayMessage renders the generated commit message with styling.
func DisplayMessage(msg string) {
	lipgloss.Println(HeaderStyle.Render("Generated commit message:"))
	lipgloss.Println(MessageStyle.Render(msg))
}

// DisplayError renders a styled error message to stderr.
func DisplayError(err error) {
	lipgloss.Fprintln(os.Stderr, ErrorStyle.Render("Error: "+err.Error()))
}

// DisplayWarning renders a styled warning message to stderr.
func DisplayWarning(msg string) {
	lipgloss.Fprintln(os.Stderr, WarningStyle.Render("Warning: "+msg))
}

// DisplayCommitResult renders the git commit output with a success indicator.
func DisplayCommitResult(output string) {
	fmt.Println(SuccessStyle.Render("✓ Committed successfully"))
	fmt.Println(output)
}

// DisplayModels renders the list of available models.
func DisplayModels(models []provider.Model) {
	fmt.Println(HeaderStyle.Render("Available models:"))
	faint := lipgloss.NewStyle().Faint(true)
	for _, m := range models {
		fmt.Printf("  %s  %s\n", m.ID, faint.Render(m.Name))
	}
}
