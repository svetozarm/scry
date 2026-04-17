package ui

import "charm.land/huh/v2"

// Action represents the user's choice after viewing a generated commit message.
type Action int

const (
	ActionAccept     Action = iota
	ActionRegenerate
	ActionCancel
)

// PromptAction displays an interactive select prompt for the user to choose
// whether to accept, regenerate, or cancel the commit message.
func PromptAction() (Action, error) {
	var action Action
	err := huh.NewSelect[Action]().
		Title("What would you like to do?").
		Options(
			huh.NewOption("Accept and commit", ActionAccept),
			huh.NewOption("Regenerate", ActionRegenerate),
			huh.NewOption("Cancel", ActionCancel),
		).
		Value(&action).
		Run()
	return action, err
}
