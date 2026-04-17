//go:build integration

package integration

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNonInteractive_RawMessageOnly(t *testing.T) {
	dir := setupGitRepo(t)
	cfgPath := writeConfig(t, dir, "provider: test\nmodel_id: test-model\n")

	stdout, _, exitCode := run(t, dir,
		[]string{"TEST_PROVIDER_RESPONSE=fix: update dependencies"},
		"--non-interactive", "--config", cfgPath,
	)

	assert.Equal(t, 0, exitCode)
	assert.Equal(t, "fix: update dependencies\n", stdout, "stdout should contain only the raw message")
}

func TestNonInteractive_NoANSICodes(t *testing.T) {
	dir := setupGitRepo(t)
	cfgPath := writeConfig(t, dir, "provider: test\nmodel_id: test-model\n")

	stdout, stderr, exitCode := run(t, dir, nil, "--non-interactive", "--config", cfgPath)

	assert.Equal(t, 0, exitCode)
	assert.NotContains(t, stdout, "\x1b[", "stdout must not contain ANSI escape codes")
	assert.NotContains(t, stderr, "\x1b[", "stderr must not contain ANSI escape codes")
}

func TestNonInteractive_NoGitCommitCreated(t *testing.T) {
	dir := setupGitRepo(t)
	cfgPath := writeConfig(t, dir, "provider: test\nmodel_id: test-model\n")

	_, _, exitCode := run(t, dir, nil, "--non-interactive", "--config", cfgPath)
	assert.Equal(t, 0, exitCode)

	log := exec.Command("git", "log", "--oneline")
	log.Dir = dir
	out, err := log.CombinedOutput()
	if err == nil {
		assert.Empty(t, strings.TrimSpace(string(out)), "no git commit should have been created")
	}
}

func TestNonInteractive_ExitCodeZero(t *testing.T) {
	dir := setupGitRepo(t)
	cfgPath := writeConfig(t, dir, "provider: test\nmodel_id: test-model\n")

	_, _, exitCode := run(t, dir, nil, "--non-interactive", "--config", cfgPath)
	assert.Equal(t, 0, exitCode)
}

func TestNonInteractive_StagedChangesPreserved(t *testing.T) {
	dir := setupGitRepo(t)
	cfgPath := writeConfig(t, dir, "provider: test\nmodel_id: test-model\n")

	_, _, exitCode := run(t, dir, nil, "--non-interactive", "--config", cfgPath)
	require.Equal(t, 0, exitCode)

	// Staged changes should still be present after non-interactive run
	diff := exec.Command("git", "diff", "--cached", "--name-only")
	diff.Dir = dir
	out, err := diff.CombinedOutput()
	require.NoError(t, err)
	assert.Contains(t, strings.TrimSpace(string(out)), "file.txt", "staged changes should be preserved")
}
