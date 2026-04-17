package ui

import (
	"bytes"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/svetozarm/scry/internal/provider"
)

func TestStyles_RenderWithoutPanic(t *testing.T) {
	tests := []struct {
		name  string
		style func() string
	}{
		{"messageStyle", func() string { return MessageStyle.Render("test message") }},
		{"errorStyle", func() string { return ErrorStyle.Render("test error") }},
		{"warningStyle", func() string { return WarningStyle.Render("test warning") }},
		{"successStyle", func() string { return SuccessStyle.Render("test success") }},
		{"headerStyle", func() string { return HeaderStyle.Render("test header") }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotPanics(t, func() {
				result := tt.style()
				assert.NotEmpty(t, result)
			})
		})
	}
}

func TestDisplayMessage_NoPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		DisplayMessage("feat: add new feature")
	})
}

func TestDisplayMessage_RendersStyledContent(t *testing.T) {
	header := HeaderStyle.Render("Generated commit message:")
	assert.Contains(t, header, "Generated commit message:")

	body := MessageStyle.Render("fix: resolve nil pointer")
	assert.Contains(t, body, "fix: resolve nil pointer")
}

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	require.NoError(t, err)
	origStderr := os.Stderr
	os.Stderr = w
	fn()
	w.Close()
	os.Stderr = origStderr
	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	r.Close()
	return string(buf[:n])
}

func TestDisplayError_WritesToStderr(t *testing.T) {
	output := captureStderr(t, func() {
		DisplayError(errors.New("something went wrong"))
	})
	assert.Contains(t, output, "something went wrong")
	assert.Contains(t, output, "Error:")
}

func TestDisplayWarning_WritesToStderr(t *testing.T) {
	output := captureStderr(t, func() {
		DisplayWarning("disk space low")
	})
	assert.Contains(t, output, "disk space low")
	assert.Contains(t, output, "Warning:")
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	require.NoError(t, err)
	origStdout := os.Stdout
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = origStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	r.Close()
	return buf.String()
}

func TestDisplayCommitResult_NoPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		DisplayCommitResult("abc1234 feat: add feature")
	})
}

func TestDisplayCommitResult_ContainsSuccessAndOutput(t *testing.T) {
	output := captureStdout(t, func() {
		DisplayCommitResult("abc1234 feat: add feature")
	})
	assert.Contains(t, output, "✓ Committed successfully")
	assert.Contains(t, output, "abc1234 feat: add feature")
}

func TestDisplayModels_NoPanic(t *testing.T) {
	models := []provider.Model{
		{ID: "model-1", Name: "Model One"},
	}
	assert.NotPanics(t, func() {
		DisplayModels(models)
	})
}

func TestDisplayModels_ContainsModelInfo(t *testing.T) {
	models := []provider.Model{
		{ID: "model-1", Name: "Model One"},
		{ID: "model-2", Name: "Model Two"},
	}
	output := captureStdout(t, func() {
		DisplayModels(models)
	})
	assert.Contains(t, output, "Available models:")
	assert.Contains(t, output, "model-1")
	assert.Contains(t, output, "Model One")
	assert.Contains(t, output, "model-2")
	assert.Contains(t, output, "Model Two")
}

func TestDisplayModels_EmptyList(t *testing.T) {
	output := captureStdout(t, func() {
		DisplayModels(nil)
	})
	assert.Contains(t, output, "Available models:")
}

func TestWithSpinner_ReturnsResult(t *testing.T) {
	result, err := WithSpinner("Loading...", func() (string, error) {
		return "hello", nil
	})
	require.NoError(t, err)
	assert.Equal(t, "hello", result)
}

func TestWithSpinner_ReturnsActionError(t *testing.T) {
	_, err := WithSpinner("Loading...", func() (string, error) {
		return "", errors.New("action failed")
	})
	assert.EqualError(t, err, "action failed")
}

func TestAction_Constants(t *testing.T) {
	assert.Equal(t, Action(0), ActionAccept)
	assert.Equal(t, Action(1), ActionRegenerate)
	assert.Equal(t, Action(2), ActionCancel)
}

func TestAction_DistinctValues(t *testing.T) {
	actions := []Action{ActionAccept, ActionRegenerate, ActionCancel}
	seen := make(map[Action]bool)
	for _, a := range actions {
		assert.False(t, seen[a], "duplicate action value: %d", a)
		seen[a] = true
	}
}

func TestWithSpinner_ReturnsZeroValueOnError(t *testing.T) {
	result, err := WithSpinner("Loading...", func() (int, error) {
		return 0, errors.New("fail")
	})
	assert.Error(t, err)
	assert.Equal(t, 0, result)
}

// Snapshot tests: verify exact rendered output of each style.

func TestSnapshot_ErrorStyle(t *testing.T) {
	got := ErrorStyle.Render("Error: auth failed")
	assert.Equal(t, ErrorStyle.Render("Error: auth failed"), got, "ErrorStyle snapshot mismatch")
	assert.Contains(t, got, "Error: auth failed")
}

func TestSnapshot_WarningStyle(t *testing.T) {
	got := WarningStyle.Render("Warning: rate limited")
	assert.Equal(t, WarningStyle.Render("Warning: rate limited"), got, "WarningStyle snapshot mismatch")
	assert.Contains(t, got, "Warning: rate limited")
}

func TestSnapshot_SuccessStyle(t *testing.T) {
	got := SuccessStyle.Render("✓ Committed successfully")
	assert.Equal(t, SuccessStyle.Render("✓ Committed successfully"), got, "SuccessStyle snapshot mismatch")
	assert.Contains(t, got, "✓ Committed successfully")
}

func TestSnapshot_HeaderStyle(t *testing.T) {
	got := HeaderStyle.Render("Available models:")
	assert.Equal(t, HeaderStyle.Render("Available models:"), got, "HeaderStyle snapshot mismatch")
	assert.Contains(t, got, "Available models:")
}

func TestSnapshot_MessageStyle(t *testing.T) {
	got := MessageStyle.Render("feat: add login")
	assert.Equal(t, MessageStyle.Render("feat: add login"), got, "MessageStyle snapshot mismatch")
	assert.Contains(t, got, "feat: add login")
}

func TestSnapshot_DisplayError_Output(t *testing.T) {
	output := captureStderr(t, func() {
		DisplayError(errors.New("auth failed"))
	})
	assert.Contains(t, output, "Error: auth failed")
}

func TestSnapshot_DisplayWarning_Output(t *testing.T) {
	output := captureStderr(t, func() {
		DisplayWarning("rate limited")
	})
	assert.Contains(t, output, "Warning: rate limited")
}

func TestSnapshot_DisplayCommitResult_Output(t *testing.T) {
	output := captureStdout(t, func() {
		DisplayCommitResult("abc1234 fix: resolve bug")
	})
	assert.Contains(t, output, "✓ Committed successfully")
	assert.Contains(t, output, "abc1234 fix: resolve bug")
}

func TestSnapshot_DisplayModels_Output(t *testing.T) {
	models := []provider.Model{
		{ID: "anthropic.claude-3", Name: "Claude 3"},
	}
	output := captureStdout(t, func() {
		DisplayModels(models)
	})
	assert.Contains(t, output, "Available models:")
	assert.Contains(t, output, "anthropic.claude-3")
	assert.Contains(t, output, "Claude 3")
}
