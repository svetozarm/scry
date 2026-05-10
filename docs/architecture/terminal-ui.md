# Module: Terminal UI (`internal/ui/`)

## Responsibility

All terminal rendering: styled output, spinner during LLM inference, progress bar for per-file summarization, interactive prompts, error display, and non-interactive plain output mode.

## Package Structure

```
internal/ui/
  ui.go          # Public API: DisplayMessage, DisplayError, DisplayWarning, DisplayProgressBar, DisplayCommitResult, DisplayModels
  spinner.go     # WithSpinner helper
  prompt.go      # PromptAction (accept/regenerate/cancel)
  prompter.go    # Prompter interface and DefaultPrompter for testability
  styles.go      # Lip Gloss style definitions
  ui_test.go     # Unit tests
```

## Public API

```go
// DisplayMessage renders the generated commit message with styling.
func DisplayMessage(msg string)

// DisplayError renders a styled error message to stderr.
func DisplayError(err error)

// DisplayWarning renders a styled warning (e.g., truncation notice).
func DisplayWarning(msg string)

// DisplayProgressBar renders an inline progress bar showing per-file summary progress.
func DisplayProgressBar(completed, total int, lastFile string)

// DisplayCommitResult renders the git commit output with a success indicator.
func DisplayCommitResult(output string)

// DisplayModels renders the model list with styling.
func DisplayModels(models []provider.Model)

// WithSpinner runs fn while displaying an inline spinner. Returns fn's result.
func WithSpinner[T any](message string, fn func() (T, error)) (T, error)

// PromptAction shows accept/regenerate/cancel and returns the user's choice.
type Action int
const (
    ActionAccept Action = iota
    ActionRegenerate
    ActionCancel
)
func PromptAction() (Action, error)
```

## Prompter Interface

The `Prompter` interface abstracts interactive UI operations for testability. The CLI layer uses this interface instead of calling UI functions directly, allowing tests to inject a mock.

```go
type Prompter interface {
    WithSpinner(message string, fn func() (string, error)) (string, error)
    PromptAction() (Action, error)
    DisplayMessage(msg string)
    DisplayCommitResult(output string)
}

// DefaultPrompter delegates to the real terminal UI functions.
type DefaultPrompter struct{}
```

## Non-Interactive Mode

When non-interactive mode is active, the UI module is bypassed entirely by the CLI layer. The CLI writes the raw message to stdout via `fmt.Fprintln` — no Charm libraries involved. Progress and warnings go to stderr as plain text.

## Styling (Lip Gloss)

- Commit message: bordered box with rounded border, accent color (color 39)
- Headers: bold, accent color (color 39)
- Errors: red foreground (color 196), bold
- Warnings: yellow/orange foreground (color 214)
- Success: green foreground (color 82)
- Model list: header + faint-styled model names
- Progress bar: inline `█░` bar with file count and last completed filename

## Interactive Prompt (Huh)

Uses `huh.NewSelect` for the accept/regenerate/cancel choice:

```go
huh.NewSelect[Action]().
    Title("What would you like to do?").
    Options(
        huh.NewOption("Accept and commit", ActionAccept),
        huh.NewOption("Regenerate", ActionRegenerate),
        huh.NewOption("Cancel", ActionCancel),
    )
```

## Dependencies

- `charm.land/lipgloss/v2`
- `charm.land/huh/v2`
- `charm.land/huh/v2/spinner`
