package ui

// Prompter abstracts interactive UI operations for testability.
type Prompter interface {
	WithSpinner(message string, fn func() (string, error)) (string, error)
	PromptAction() (Action, error)
	DisplayMessage(msg string)
	DisplayCommitResult(output string)
}

// DefaultPrompter uses the real terminal UI functions.
type DefaultPrompter struct{}

func (DefaultPrompter) WithSpinner(message string, fn func() (string, error)) (string, error) {
	return WithSpinner(message, fn)
}

func (DefaultPrompter) PromptAction() (Action, error) {
	return PromptAction()
}

func (DefaultPrompter) DisplayMessage(msg string) {
	DisplayMessage(msg)
}

func (DefaultPrompter) DisplayCommitResult(output string) {
	DisplayCommitResult(output)
}
