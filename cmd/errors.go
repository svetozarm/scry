package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/svetozarm/scry/internal/config"
	"github.com/svetozarm/scry/internal/git"
	"github.com/svetozarm/scry/internal/provider"
	"github.com/svetozarm/scry/internal/ui"
)

// silentError wraps an error so Cobra won't print it again.
type silentError struct {
	err      error
	exitCode int
}

func (e *silentError) Error() string { return e.err.Error() }
func (e *silentError) Unwrap() error { return e.err }

// ExitCode returns the process exit code for this error.
func (e *silentError) ExitCode() int { return e.exitCode }

type errorMapping struct {
	msg  string
	code int
}

func mapError(err error) errorMapping {
	switch {
	case errors.Is(err, git.ErrNoRepo):
		return errorMapping{"not inside a git repository", 2}
	case errors.Is(err, git.ErrNoStagedChanges):
		return errorMapping{"no staged changes — stage files with git add first", 2}
	case errors.As(err, new(*git.ErrCommitFailed)):
		return errorMapping{err.Error(), 2}
	case errors.Is(err, provider.ErrAuth):
		return errorMapping{"authentication/authorisation failed — check your credentials", 3}
	case errors.Is(err, provider.ErrRateLimit):
		return errorMapping{"rate limit exceeded — please retry later", 3}
	case errors.Is(err, provider.ErrTimeout):
		return errorMapping{"request timed out — the LLM provider did not respond in time", 3}
	case errors.Is(err, provider.ErrModelNotFound):
		return errorMapping{"model not found — check your model_id in config", 3}
	case errors.As(err, new(*config.ConfigParseError)):
		return errorMapping{err.Error(), 4}
	default:
		return errorMapping{sanitizeError(err), 1}
	}
}

// sanitizeError strips potential credentials from error messages.
func sanitizeError(err error) string {
	msg := err.Error()
	if strings.Contains(msg, "credential") || strings.Contains(msg, "AKIA") || strings.Contains(msg, "secret") {
		return "an unexpected error occurred (details redacted for security)"
	}
	return msg
}

func handleError(err error) error {
	m := mapError(err)
	if nonInteractive {
		fmt.Fprintln(os.Stderr, "Error: "+m.msg)
	} else {
		ui.DisplayError(fmt.Errorf("%s", m.msg))
	}
	return &silentError{err: err, exitCode: m.code}
}
