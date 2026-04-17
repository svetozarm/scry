//go:build integration

package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestError_NoGitRepo(t *testing.T) {
	dir := t.TempDir()
	cfgPath := writeConfig(t, dir, "provider: test\nmodel_id: m\n")

	_, stderr, exitCode := run(t, dir, nil, "--non-interactive", "--config", cfgPath)

	assert.NotEqual(t, 0, exitCode)
	assert.Contains(t, stderr, "not inside a git repository")
}

func TestError_NoStagedChanges(t *testing.T) {
	dir := t.TempDir()
	cfgPath := writeConfig(t, dir, "provider: test\nmodel_id: m\n")

	// Init a git repo but don't stage anything
	for _, args := range [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
	} {
		c := exec.Command(args[0], args[1:]...)
		c.Dir = dir
		out, err := c.CombinedOutput()
		require.NoError(t, err, string(out))
	}
	require.NoError(t, os.WriteFile(filepath.Join(dir, "file.txt"), []byte("hello"), 0644))

	cmd := exec.Command(binaryPath, "--non-interactive", "--config", cfgPath)
	cmd.Dir = dir
	cmd.Env = os.Environ()
	var errBuf strings.Builder
	cmd.Stderr = &errBuf
	err := cmd.Run()

	assert.Error(t, err)
	assert.Contains(t, errBuf.String(), "no staged changes")
}

func TestError_InvalidConfig(t *testing.T) {
	dir := setupGitRepo(t)
	cfgDir := filepath.Join(dir, ".scry")
	require.NoError(t, os.MkdirAll(cfgDir, 0755))
	cfgPath := filepath.Join(cfgDir, "config.yaml")
	require.NoError(t, os.WriteFile(cfgPath, []byte("{{invalid yaml"), 0644))

	_, stderr, exitCode := run(t, dir, nil, "--non-interactive", "--config", cfgPath)

	assert.NotEqual(t, 0, exitCode)
	assert.Contains(t, stderr, cfgPath)
}

func TestError_AuthError(t *testing.T) {
	dir := setupGitRepo(t)
	cfgPath := writeConfig(t, dir, "provider: test\nmodel_id: m\n")

	_, stderr, exitCode := run(t, dir,
		[]string{"TEST_PROVIDER_ERROR=auth"},
		"--non-interactive", "--config", cfgPath,
	)

	assert.NotEqual(t, 0, exitCode)
	assert.Contains(t, stderr, "authentication")
}

func TestError_RateLimit(t *testing.T) {
	dir := setupGitRepo(t)
	cfgPath := writeConfig(t, dir, "provider: test\nmodel_id: m\n")

	_, stderr, exitCode := run(t, dir,
		[]string{"TEST_PROVIDER_ERROR=ratelimit"},
		"--non-interactive", "--config", cfgPath,
	)

	assert.NotEqual(t, 0, exitCode)
	assert.Contains(t, stderr, "retry later")
}

func TestError_Timeout(t *testing.T) {
	dir := setupGitRepo(t)
	cfgPath := writeConfig(t, dir, "provider: test\nmodel_id: m\n")

	_, stderr, exitCode := run(t, dir,
		[]string{"TEST_PROVIDER_ERROR=timeout"},
		"--non-interactive", "--config", cfgPath,
	)

	assert.NotEqual(t, 0, exitCode)
	assert.Contains(t, stderr, "timed out")
}

func TestError_ModelNotFound(t *testing.T) {
	dir := setupGitRepo(t)
	cfgPath := writeConfig(t, dir, "provider: test\nmodel_id: nonexistent-model\n")

	_, stderr, exitCode := run(t, dir,
		[]string{"TEST_PROVIDER_ERROR=model"},
		"--non-interactive", "--config", cfgPath,
	)

	assert.NotEqual(t, 0, exitCode)
	assert.Contains(t, stderr, "model not found")
}
