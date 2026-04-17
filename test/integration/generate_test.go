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

var binaryPath string

func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "scry-e2e-*")
	if err != nil {
		panic(err)
	}
	binaryPath = filepath.Join(tmp, "scry")

	build := exec.Command("go", "build", "-tags=integration", "-o", binaryPath, ".")
	build.Dir = projectRoot()
	build.Env = append(os.Environ(), "CGO_ENABLED=0")
	out, err := build.CombinedOutput()
	if err != nil {
		panic("build failed: " + string(out))
	}

	code := m.Run()
	os.RemoveAll(tmp)
	os.Exit(code)
}

func projectRoot() string {
	// test/integration -> project root
	wd, _ := os.Getwd()
	return filepath.Join(wd, "..", "..")
}

func setupGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
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
	c := exec.Command("git", "add", "file.txt")
	c.Dir = dir
	require.NoError(t, c.Run())
	return dir
}

func writeConfig(t *testing.T, dir, content string) string {
	t.Helper()
	cfgDir := filepath.Join(dir, ".scry")
	require.NoError(t, os.MkdirAll(cfgDir, 0755))
	p := filepath.Join(cfgDir, "config.yaml")
	require.NoError(t, os.WriteFile(p, []byte(content), 0644))
	return p
}

func run(t *testing.T, dir string, env []string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	exitCode = 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			exitCode = ee.ExitCode()
		} else {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	return outBuf.String(), errBuf.String(), exitCode
}

func TestGenerate_HappyPath(t *testing.T) {
	dir := setupGitRepo(t)
	cfgPath := writeConfig(t, dir, "provider: test\nmodel_id: test-model\n")

	stdout, _, exitCode := run(t, dir,
		[]string{"TEST_PROVIDER_RESPONSE=feat: add file.txt"},
		"--non-interactive", "--config", cfgPath,
	)

	assert.Equal(t, 0, exitCode)
	assert.Equal(t, "feat: add file.txt\n", stdout)
}

func TestGenerate_CustomResponse(t *testing.T) {
	dir := setupGitRepo(t)
	cfgPath := writeConfig(t, dir, "provider: test\nmodel_id: m\n")

	stdout, _, exitCode := run(t, dir,
		[]string{"TEST_PROVIDER_RESPONSE=fix: resolve null pointer"},
		"--non-interactive", "--config", cfgPath,
	)

	assert.Equal(t, 0, exitCode)
	assert.Contains(t, stdout, "fix: resolve null pointer")
}

func TestGenerate_DefaultResponse(t *testing.T) {
	dir := setupGitRepo(t)
	cfgPath := writeConfig(t, dir, "provider: test\nmodel_id: m\n")

	stdout, _, exitCode := run(t, dir, nil, "--non-interactive", "--config", cfgPath)

	assert.Equal(t, 0, exitCode)
	assert.Equal(t, "feat: add new feature\n", stdout)
}

func TestGenerate_NoGitCommitCreated(t *testing.T) {
	dir := setupGitRepo(t)
	cfgPath := writeConfig(t, dir, "provider: test\nmodel_id: m\n")

	_, _, exitCode := run(t, dir, nil, "--non-interactive", "--config", cfgPath)
	assert.Equal(t, 0, exitCode)

	// Verify no commit was created
	log := exec.Command("git", "log", "--oneline")
	log.Dir = dir
	out, err := log.CombinedOutput()
	// Either fails (no commits) or output is empty
	if err == nil {
		assert.Empty(t, strings.TrimSpace(string(out)))
	}
}

func TestGenerate_StdoutHasNoANSI(t *testing.T) {
	dir := setupGitRepo(t)
	cfgPath := writeConfig(t, dir, "provider: test\nmodel_id: m\n")

	stdout, _, _ := run(t, dir, nil, "--non-interactive", "--config", cfgPath)
	assert.NotContains(t, stdout, "\x1b[", "stdout must not contain ANSI escape codes")
}

func TestListModels_HappyPath(t *testing.T) {
	dir := t.TempDir()
	cfgPath := writeConfig(t, dir, "provider: test\n")

	stdout, _, exitCode := run(t, dir, nil, "list-models", "--non-interactive", "--config", cfgPath)

	assert.Equal(t, 0, exitCode)
	assert.Contains(t, stdout, "test-model-1\tTest Model 1")
	assert.Contains(t, stdout, "test-model-2\tTest Model 2")
}

func TestListModels_ExitCodeZero(t *testing.T) {
	dir := t.TempDir()
	cfgPath := writeConfig(t, dir, "provider: test\n")

	_, _, exitCode := run(t, dir, nil, "list-models", "--non-interactive", "--config", cfgPath)

	assert.Equal(t, 0, exitCode)
}

func TestListModels_StdoutHasNoANSI(t *testing.T) {
	dir := t.TempDir()
	cfgPath := writeConfig(t, dir, "provider: test\n")

	stdout, _, _ := run(t, dir, nil, "list-models", "--non-interactive", "--config", cfgPath)

	assert.NotContains(t, stdout, "\x1b[", "stdout must not contain ANSI escape codes")
}
