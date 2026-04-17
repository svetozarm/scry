package cmd

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/svetozarm/scry/internal/provider"
	"github.com/svetozarm/scry/internal/ui"
)

// mockPrompter implements ui.Prompter for testing the interactive loop.
type mockPrompter struct {
	actions       []ui.Action // sequence of actions to return
	actionIndex   int
	messages      []string // messages displayed
	commitOutputs []string // commit results displayed
	spinnerCalls  int
}

func (m *mockPrompter) WithSpinner(_ string, fn func() (string, error)) (string, error) {
	m.spinnerCalls++
	return fn()
}

func (m *mockPrompter) PromptAction() (ui.Action, error) {
	a := m.actions[m.actionIndex]
	m.actionIndex++
	return a, nil
}

func (m *mockPrompter) DisplayMessage(msg string) {
	m.messages = append(m.messages, msg)
}

func (m *mockPrompter) DisplayCommitResult(output string) {
	m.commitOutputs = append(m.commitOutputs, output)
}

// mockProvider implements provider.Provider for testing.
type mockProvider struct {
	invokeResult string
	invokeErr    error
	models       []provider.Model
	modelsErr    error
	maxTok       int
}

func (m *mockProvider) Invoke(_ context.Context, _ string, _ string) (string, error) {
	return m.invokeResult, m.invokeErr
}

func (m *mockProvider) ListModels(_ context.Context) ([]provider.Model, error) {
	return m.models, m.modelsErr
}

func (m *mockProvider) MaxTokens(_ string) int {
	if m.maxTok > 0 {
		return m.maxTok
	}
	return 128000
}

// setupGitRepo creates a temp git repo with a staged file and a config file.
func setupGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test User"},
	}
	for _, args := range cmds {
		c := exec.Command(args[0], args[1:]...)
		c.Dir = dir
		out, err := c.CombinedOutput()
		require.NoError(t, err, "git setup failed: %s", string(out))
	}

	// Create and stage a file
	require.NoError(t, os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hello"), 0644))
	c := exec.Command("git", "add", "hello.txt")
	c.Dir = dir
	require.NoError(t, c.Run())

	// Create config pointing to mock provider
	cfgDir := filepath.Join(dir, ".scry")
	require.NoError(t, os.MkdirAll(cfgDir, 0755))
	cfgContent := "provider: mock\nmodel_id: test-model\nprompt: \"Generate a commit message for:\\n{{branch_name}}\"\n"
	require.NoError(t, os.WriteFile(filepath.Join(cfgDir, "config.yaml"), []byte(cfgContent), 0644))

	return dir
}

func registerMockProvider(mock *mockProvider) {
	provider.Register("mock", func(_ map[string]string) (provider.Provider, error) {
		return mock, nil
	})
}

// assertSilentError checks that the error is a silentError with the expected exit code.
func assertSilentError(t *testing.T, err error, wantCode int) {
	t.Helper()
	var se *silentError
	require.True(t, errors.As(err, &se), "expected silentError, got %T", err)
	assert.Equal(t, wantCode, se.ExitCode())
}

// runNonInteractive runs the root command in non-interactive mode, capturing stderr.
// Returns the error and captured stderr output.
func runNonInteractive(t *testing.T, dir string) (error, string) {
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

	rootCmd.SetArgs([]string{"--non-interactive", "--config", filepath.Join(dir, ".scry", "config.yaml")})
	execErr := rootCmd.Execute()

	w.Close()
	os.Stderr = origStderr
	buf := make([]byte, 4096)
	n, _ := r.Read(buf)

	return execErr, string(buf[:n])
}

func TestRunGenerate_NonInteractive_HappyPath(t *testing.T) {
	mock := &mockProvider{invokeResult: "feat: add hello.txt"}
	registerMockProvider(mock)

	dir := setupGitRepo(t)

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetArgs([]string{"--non-interactive", "--config", filepath.Join(dir, ".scry", "config.yaml")})

	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	defer os.Chdir(origDir)

	err := rootCmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "feat: add hello.txt")
}

func TestRunGenerate_NoRepo(t *testing.T) {
	dir := t.TempDir()

	cfgDir := filepath.Join(dir, ".scry")
	require.NoError(t, os.MkdirAll(cfgDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(cfgDir, "config.yaml"), []byte("provider: mock\n"), 0644))

	err, stderr := runNonInteractive(t, dir)
	assertSilentError(t, err, 2)
	assert.Contains(t, stderr, "not inside a git repository")
}

func TestRunGenerate_NoStagedChanges(t *testing.T) {
	dir := t.TempDir()

	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test User"},
	}
	for _, args := range cmds {
		c := exec.Command(args[0], args[1:]...)
		c.Dir = dir
		require.NoError(t, c.Run())
	}

	cfgDir := filepath.Join(dir, ".scry")
	require.NoError(t, os.MkdirAll(cfgDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(cfgDir, "config.yaml"), []byte("provider: mock\n"), 0644))

	err, stderr := runNonInteractive(t, dir)
	assertSilentError(t, err, 2)
	assert.Contains(t, stderr, "no staged changes")
}

func TestRunGenerate_ProviderAuthError(t *testing.T) {
	mock := &mockProvider{invokeErr: provider.ErrAuth}
	registerMockProvider(mock)

	dir := setupGitRepo(t)
	err, stderr := runNonInteractive(t, dir)
	assertSilentError(t, err, 3)
	assert.Contains(t, stderr, "authentication/authorisation failed")
}

func TestRunGenerate_ProviderRateLimitError(t *testing.T) {
	mock := &mockProvider{invokeErr: provider.ErrRateLimit}
	registerMockProvider(mock)

	dir := setupGitRepo(t)
	err, stderr := runNonInteractive(t, dir)
	assertSilentError(t, err, 3)
	assert.Contains(t, stderr, "rate limit exceeded")
}

func TestRunGenerate_ProviderTimeoutError(t *testing.T) {
	mock := &mockProvider{invokeErr: provider.ErrTimeout}
	registerMockProvider(mock)

	dir := setupGitRepo(t)
	err, stderr := runNonInteractive(t, dir)
	assertSilentError(t, err, 3)
	assert.Contains(t, stderr, "request timed out")
}

func TestRunGenerate_ProviderModelNotFoundError(t *testing.T) {
	mock := &mockProvider{invokeErr: provider.ErrModelNotFound}
	registerMockProvider(mock)

	dir := setupGitRepo(t)
	err, stderr := runNonInteractive(t, dir)
	assertSilentError(t, err, 3)
	assert.Contains(t, stderr, "model not found")
}

func TestRunGenerate_Interactive_RegenerateThenAccept(t *testing.T) {
	mock := &mockProvider{invokeResult: "feat: add hello.txt"}
	registerMockProvider(mock)

	mp := &mockPrompter{
		actions: []ui.Action{ui.ActionRegenerate, ui.ActionAccept},
	}
	origPrompter := prompter
	prompter = mp
	defer func() { prompter = origPrompter }()

	dir := setupGitRepo(t)
	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	defer os.Chdir(origDir)

	nonInteractive = false
	rootCmd.SetArgs([]string{"--config", filepath.Join(dir, ".scry", "config.yaml")})

	err := rootCmd.Execute()
	require.NoError(t, err)

	assert.Equal(t, 2, mp.spinnerCalls)
	assert.Len(t, mp.messages, 2)
	assert.Len(t, mp.commitOutputs, 1)
}

func TestRunGenerate_InvalidYAMLConfig(t *testing.T) {
	dir := t.TempDir()

	// Set up a git repo with staged changes so we get past git checks
	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test User"},
	}
	for _, args := range cmds {
		c := exec.Command(args[0], args[1:]...)
		c.Dir = dir
		require.NoError(t, c.Run())
	}
	require.NoError(t, os.WriteFile(filepath.Join(dir, "f.txt"), []byte("x"), 0644))
	c := exec.Command("git", "add", "f.txt")
	c.Dir = dir
	require.NoError(t, c.Run())

	// Write invalid YAML config
	cfgPath := filepath.Join(dir, "bad.yaml")
	require.NoError(t, os.WriteFile(cfgPath, []byte(":\ninvalid: [yaml\n"), 0644))

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
	stderr := string(buf[:n])

	assertSilentError(t, execErr, 4)
	assert.Contains(t, stderr, "bad.yaml")
}

func TestRunGenerate_GitCommitFailure(t *testing.T) {
	mock := &mockProvider{invokeResult: "feat: test commit"}
	registerMockProvider(mock)

	dir := setupGitRepo(t)

	// Install a pre-commit hook that always fails
	hooksDir := filepath.Join(dir, ".git", "hooks")
	require.NoError(t, os.MkdirAll(hooksDir, 0755))
	hookScript := "#!/bin/sh\necho 'hook rejected'\nexit 1\n"
	require.NoError(t, os.WriteFile(filepath.Join(hooksDir, "pre-commit"), []byte(hookScript), 0755))

	mp := &mockPrompter{actions: []ui.Action{ui.ActionAccept}}
	origPrompter := prompter
	prompter = mp
	defer func() { prompter = origPrompter }()

	nonInteractive = false

	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	defer os.Chdir(origDir)

	r, w, err := os.Pipe()
	require.NoError(t, err)
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

func TestRunGenerate_DiffTruncationWarning(t *testing.T) {
	mock := &mockProvider{invokeResult: "fix: truncated", maxTok: 20}
	registerMockProvider(mock)

	dir := setupGitRepo(t)

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)

	nonInteractive = true
	defer func() { nonInteractive = false }()

	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	defer os.Chdir(origDir)

	r, w, err := os.Pipe()
	require.NoError(t, err)
	origStderr := os.Stderr
	os.Stderr = w

	rootCmd.SetArgs([]string{"--non-interactive", "--config", filepath.Join(dir, ".scry", "config.yaml")})
	execErr := rootCmd.Execute()

	w.Close()
	os.Stderr = origStderr
	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	stderr := string(buf[:n])

	// Truncation is a warning, not an error — exit code should be 0
	require.NoError(t, execErr)
	assert.Contains(t, stderr, "truncated")
	assert.Contains(t, stdout.String(), "fix: truncated")
}

func TestRunGenerate_Interactive_Cancel(t *testing.T) {
	mock := &mockProvider{invokeResult: "feat: add hello.txt"}
	registerMockProvider(mock)

	mp := &mockPrompter{
		actions: []ui.Action{ui.ActionCancel},
	}
	origPrompter := prompter
	prompter = mp
	defer func() { prompter = origPrompter }()

	dir := setupGitRepo(t)
	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	defer os.Chdir(origDir)

	nonInteractive = false
	rootCmd.SetArgs([]string{"--config", filepath.Join(dir, ".scry", "config.yaml")})

	err := rootCmd.Execute()
	require.NoError(t, err)

	assert.Equal(t, 1, mp.spinnerCalls)
	assert.Len(t, mp.messages, 1)
	assert.Empty(t, mp.commitOutputs)
}
