package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/svetozarm/scry/internal/provider"
)

// setupNonInteractive prepares a git repo, registers a mock provider, and
// configures the root command for non-interactive execution. Returns stdout
// buffer and cleanup function.
func setupNonInteractive(t *testing.T, mock *mockProvider, cfgOverride string) (*bytes.Buffer, string) {
	t.Helper()
	registerMockProvider(mock)
	dir := setupGitRepo(t)

	if cfgOverride != "" {
		cfgPath := filepath.Join(dir, ".scry", "config.yaml")
		require.NoError(t, os.WriteFile(cfgPath, []byte(cfgOverride), 0644))
	}

	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { os.Chdir(origDir) })

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetArgs([]string{"--non-interactive", "--config", filepath.Join(dir, ".scry", "config.yaml")})

	return &stdout, dir
}

func TestNonInteractive_PlainTextOutput(t *testing.T) {
	mock := &mockProvider{invokeResult: "fix: correct typo in README"}
	stdout, _ := setupNonInteractive(t, mock, "")

	err := rootCmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Equal(t, "fix: correct typo in README\n", out)
	assert.False(t, strings.Contains(out, "\x1b["), "stdout must not contain ANSI escape codes")
}

func TestNonInteractive_NoGitCommit(t *testing.T) {
	mock := &mockProvider{invokeResult: "feat: new feature"}
	_, dir := setupNonInteractive(t, mock, "")

	err := rootCmd.Execute()
	require.NoError(t, err)

	// git log should fail because there are no commits yet
	c := exec.Command("git", "log", "--oneline", "-1")
	c.Dir = dir
	out, logErr := c.CombinedOutput()
	// Either git log fails (no commits) or the output doesn't contain our message
	if logErr == nil {
		assert.NotContains(t, string(out), "feat: new feature")
	}
}

func TestNonInteractive_ExitCodeZero(t *testing.T) {
	mock := &mockProvider{invokeResult: "chore: cleanup"}
	setupNonInteractive(t, mock, "")

	err := rootCmd.Execute()
	assert.NoError(t, err, "exit code should be 0 on success")
}

func TestNonInteractive_ExitCodeNonZeroOnError(t *testing.T) {
	mock := &mockProvider{invokeErr: provider.ErrAuth}
	setupNonInteractive(t, mock, "")

	r, w, err := os.Pipe()
	require.NoError(t, err)
	origStderr := os.Stderr
	os.Stderr = w

	execErr := rootCmd.Execute()

	w.Close()
	os.Stderr = origStderr
	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	stderr := string(buf[:n])

	assertSilentError(t, execErr, 3)
	assert.Contains(t, stderr, "authentication/authorisation failed")
}

func TestNonInteractive_TruncationWarningOnStderr(t *testing.T) {
	mock := &mockProvider{invokeResult: "fix: truncated diff", maxTok: 20}
	stdout, _ := setupNonInteractive(t, mock, "")

	r, w, err := os.Pipe()
	require.NoError(t, err)
	origStderr := os.Stderr
	os.Stderr = w

	execErr := rootCmd.Execute()

	w.Close()
	os.Stderr = origStderr
	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	stderr := string(buf[:n])

	require.NoError(t, execErr)
	assert.Contains(t, stderr, "truncated")
	assert.Equal(t, "fix: truncated diff\n", stdout.String(), "stdout should contain only the message")
}
