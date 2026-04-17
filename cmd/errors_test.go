package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/svetozarm/scry/internal/config"
	"github.com/svetozarm/scry/internal/git"
	"github.com/svetozarm/scry/internal/provider"
	"github.com/svetozarm/scry/internal/ui"
)

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
	return string(buf[:n])
}

func TestHandleError_ExitCodes(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode int
	}{
		{"ErrNoRepo", git.ErrNoRepo, 2},
		{"ErrNoStagedChanges", git.ErrNoStagedChanges, 2},
		{"ErrCommitFailed", &git.ErrCommitFailed{Output: "fail"}, 2},
		{"ErrAuth", provider.ErrAuth, 3},
		{"ErrRateLimit", provider.ErrRateLimit, 3},
		{"ErrTimeout", provider.ErrTimeout, 3},
		{"ErrModelNotFound", provider.ErrModelNotFound, 3},
		{"ConfigParseError", &config.ConfigParseError{Path: "f", Message: "bad"}, 4},
		{"UnknownError", fmt.Errorf("something broke"), 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nonInteractive = true
			defer func() { nonInteractive = false }()

			captureStderr(t, func() {
				result := handleError(tt.err)
				var se *silentError
				require.True(t, errors.As(result, &se))
				assert.Equal(t, tt.wantCode, se.exitCode)
			})
		})
	}
}

func TestHandleError_NonInteractive_PlainMessages(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		wantMsg string
	}{
		{"ErrNoRepo", git.ErrNoRepo, "not inside a git repository"},
		{"ErrNoStagedChanges", git.ErrNoStagedChanges, "no staged changes"},
		{"ErrCommitFailed", &git.ErrCommitFailed{Output: "hook failed"}, "git commit failed: hook failed"},
		{"ErrAuth", provider.ErrAuth, "authentication/authorisation failed"},
		{"ErrRateLimit", provider.ErrRateLimit, "rate limit exceeded"},
		{"ErrTimeout", provider.ErrTimeout, "request timed out"},
		{"ErrModelNotFound", provider.ErrModelNotFound, "model not found"},
		{"ConfigParseError", &config.ConfigParseError{Path: "config.yaml", Message: "bad yaml"}, "invalid config config.yaml: bad yaml"},
		{"UnknownError", fmt.Errorf("something broke"), "something broke"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nonInteractive = true
			defer func() { nonInteractive = false }()

			output := captureStderr(t, func() {
				handleError(tt.err)
			})
			assert.Contains(t, output, tt.wantMsg)
		})
	}
}

func TestHandleError_Interactive_StyledOutput(t *testing.T) {
	nonInteractive = false
	defer func() { nonInteractive = false }()

	output := captureStderr(t, func() {
		handleError(provider.ErrAuth)
	})
	// Styled output should contain the message (possibly with ANSI codes)
	assert.Contains(t, output, "authentication/authorisation failed")
}

func TestHandleError_WrapsOriginalError(t *testing.T) {
	nonInteractive = true
	defer func() { nonInteractive = false }()

	captureStderr(t, func() {
		result := handleError(provider.ErrAuth)
		assert.True(t, errors.Is(result, provider.ErrAuth))
	})
}

// ---------------------------------------------------------------------------
// CLI-level error path tests (phase-10 table, task 10.2)
// Each test sets up the error condition, runs the CLI, and asserts stderr +
// exit code.
// ---------------------------------------------------------------------------

// runCLI runs the root command in non-interactive mode from dir, returning the
// error and captured stderr.
func runCLI(t *testing.T, dir, cfgPath string) (error, string) {
	t.Helper()
	nonInteractive = true
	defer func() { nonInteractive = false }()

	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	defer os.Chdir(origDir)

	r, w, err := os.Pipe()
	require.NoError(t, err)
	origStderr := os.Stderr
	os.Stderr = w

	rootCmd.SetArgs([]string{"--non-interactive", "--config", cfgPath})
	execErr := rootCmd.Execute()

	w.Close()
	os.Stderr = origStderr
	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	return execErr, string(buf[:n])
}

func TestErrorPath_AuthError(t *testing.T) {
	registerMockProvider(&mockProvider{invokeErr: provider.ErrAuth})
	dir := setupGitRepo(t)
	err, stderr := runCLI(t, dir, filepath.Join(dir, ".scry", "config.yaml"))
	assertSilentError(t, err, 3)
	assert.Contains(t, stderr, "authentication/authorisation failed")
}

func TestErrorPath_RateLimit(t *testing.T) {
	registerMockProvider(&mockProvider{invokeErr: provider.ErrRateLimit})
	dir := setupGitRepo(t)
	err, stderr := runCLI(t, dir, filepath.Join(dir, ".scry", "config.yaml"))
	assertSilentError(t, err, 3)
	assert.Contains(t, stderr, "rate limit exceeded")
}

func TestErrorPath_Timeout(t *testing.T) {
	registerMockProvider(&mockProvider{invokeErr: provider.ErrTimeout})
	dir := setupGitRepo(t)
	err, stderr := runCLI(t, dir, filepath.Join(dir, ".scry", "config.yaml"))
	assertSilentError(t, err, 3)
	assert.Contains(t, stderr, "request timed out")
}

func TestErrorPath_ModelNotFound(t *testing.T) {
	registerMockProvider(&mockProvider{invokeErr: provider.ErrModelNotFound})
	dir := setupGitRepo(t)
	err, stderr := runCLI(t, dir, filepath.Join(dir, ".scry", "config.yaml"))
	assertSilentError(t, err, 3)
	assert.Contains(t, stderr, "model not found")
}

func TestErrorPath_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "bad.yaml")
	require.NoError(t, os.WriteFile(cfgPath, []byte(":\ninvalid: [yaml\n"), 0644))

	err, stderr := runCLI(t, dir, cfgPath)
	assertSilentError(t, err, 4)
	assert.Contains(t, stderr, "bad.yaml")
}

func TestErrorPath_NoRepo(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "cfg.yaml")
	require.NoError(t, os.WriteFile(cfgPath, []byte("provider: mock\n"), 0644))

	err, stderr := runCLI(t, dir, cfgPath)
	assertSilentError(t, err, 2)
	assert.Contains(t, stderr, "not inside a git repository")
}

func TestErrorPath_NoStagedChanges(t *testing.T) {
	dir := t.TempDir()
	for _, args := range [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
	} {
		c := exec.Command(args[0], args[1:]...)
		c.Dir = dir
		require.NoError(t, c.Run())
	}
	cfgPath := filepath.Join(dir, "cfg.yaml")
	require.NoError(t, os.WriteFile(cfgPath, []byte("provider: mock\n"), 0644))

	err, stderr := runCLI(t, dir, cfgPath)
	assertSilentError(t, err, 2)
	assert.Contains(t, stderr, "no staged changes")
}

func TestErrorPath_CommitFailure(t *testing.T) {
	registerMockProvider(&mockProvider{invokeResult: "feat: test"})
	dir := setupGitRepo(t)

	// Install a pre-commit hook that always fails
	hooksDir := filepath.Join(dir, ".git", "hooks")
	require.NoError(t, os.MkdirAll(hooksDir, 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(hooksDir, "pre-commit"),
		[]byte("#!/bin/sh\nexit 1\n"), 0755,
	))

	mp := &mockPrompter{actions: []ui.Action{ui.ActionAccept}}
	origPrompter := prompter
	prompter = mp
	defer func() { prompter = origPrompter }()

	// Run interactive (commit happens only in interactive mode)
	nonInteractive = false
	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	defer os.Chdir(origDir)

	r, w, pipeErr := os.Pipe()
	require.NoError(t, pipeErr)
	origStderr := os.Stderr
	os.Stderr = w

	rootCmd.SetArgs([]string{"--config", filepath.Join(dir, ".scry", "config.yaml")})
	execErr := rootCmd.Execute()

	w.Close()
	os.Stderr = origStderr
	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	stderr := string(buf[:n])

	assertSilentError(t, execErr, 2)
	assert.Contains(t, stderr, "git commit failed")
}

func TestErrorPath_DiffTruncationWarning(t *testing.T) {
	registerMockProvider(&mockProvider{invokeResult: "fix: truncated", maxTok: 20})
	dir := setupGitRepo(t)

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)

	err, stderr := runCLI(t, dir, filepath.Join(dir, ".scry", "config.yaml"))

	// Truncation is a warning — exit code 0
	require.NoError(t, err)
	assert.Contains(t, stderr, "truncated")
	assert.Contains(t, stdout.String(), "fix: truncated")
}

// ---------------------------------------------------------------------------
// Task 10.3: Verify no data leakage
// ---------------------------------------------------------------------------

func TestNoDataLeakage_NoFileCreated(t *testing.T) {
	mock := &mockProvider{invokeErr: provider.ErrAuth}
	registerMockProvider(mock)

	dir := setupGitRepo(t)

	// Snapshot directory contents before CLI run
	before, err := os.ReadDir(dir)
	require.NoError(t, err)
	beforeNames := map[string]bool{}
	for _, e := range before {
		beforeNames[e.Name()] = true
	}

	runCLI(t, dir, filepath.Join(dir, ".scry", "config.yaml"))

	// Check no new files were created
	after, err := os.ReadDir(dir)
	require.NoError(t, err)
	for _, e := range after {
		if !beforeNames[e.Name()] {
			t.Errorf("unexpected file created during CLI run: %s", e.Name())
		}
	}
}

func TestNoDataLeakage_StderrNoDiffContent(t *testing.T) {
	mock := &mockProvider{invokeErr: provider.ErrAuth}
	registerMockProvider(mock)

	dir := setupGitRepo(t)

	// The staged file contains "hello" — ensure it doesn't appear in stderr
	_, stderr := runCLI(t, dir, filepath.Join(dir, ".scry", "config.yaml"))
	assert.NotContains(t, stderr, "hello", "stderr should not contain diff content")
	assert.NotContains(t, stderr, "diff --git", "stderr should not contain raw diff output")
}

func TestNoDataLeakage_StderrNoCredentials(t *testing.T) {
	fakeCredErr := fmt.Errorf("credential error: AKIAIOSFODNN7EXAMPLE / wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY")
	mock := &mockProvider{invokeErr: fakeCredErr}
	registerMockProvider(mock)

	dir := setupGitRepo(t)
	_, stderr := runCLI(t, dir, filepath.Join(dir, ".scry", "config.yaml"))

	assert.NotContains(t, stderr, "AKIAIOSFODNN7EXAMPLE", "stderr should not contain AWS access key")
	assert.NotContains(t, stderr, "wJalrXUtnFEMI", "stderr should not contain AWS secret key")
}
