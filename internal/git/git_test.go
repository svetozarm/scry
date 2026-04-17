package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	return string(out)
}

func TestEnsureRepo_ValidRepo(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")

	err := EnsureRepo(dir)
	assert.NoError(t, err)
}

func TestEnsureRepo_NotARepo(t *testing.T) {
	dir := t.TempDir()

	err := EnsureRepo(dir)
	assert.ErrorIs(t, err, ErrNoRepo)
}

func TestEnsureStagedChanges_WithStagedFiles(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "file.txt"), []byte("hello"), 0644))
	runGit(t, dir, "add", "file.txt")

	err := EnsureStagedChanges(dir)
	assert.NoError(t, err)
}

func TestEnsureStagedChanges_NoStagedFiles(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")

	err := EnsureStagedChanges(dir)
	assert.ErrorIs(t, err, ErrNoStagedChanges)
}

func TestDiff_ReturnsStagedDiff(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "file.txt"), []byte("hello world"), 0644))
	runGit(t, dir, "add", "file.txt")

	diff, err := Diff(dir)
	require.NoError(t, err)
	assert.Contains(t, diff, "hello world")
}

func TestBranchName_ReturnsCurrentBranch(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init", "-b", "main")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "file.txt"), []byte("init"), 0644))
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "init")

	branch, err := BranchName(dir)
	require.NoError(t, err)
	assert.Equal(t, "main", branch)
}

func TestAuthor_ReturnsConfiguredAuthor(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.name", "Test User")

	author, err := Author(dir)
	require.NoError(t, err)
	assert.Equal(t, "Test User", author)
}

func TestCommit_Success(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init", "-b", "main")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "Test")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "file.txt"), []byte("hello"), 0644))
	runGit(t, dir, "add", "file.txt")

	output, err := Commit(dir, "initial commit")
	require.NoError(t, err)
	assert.Contains(t, output, "initial commit")
}

func TestCommit_Failure(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")

	_, err := Commit(dir, "should fail")
	require.Error(t, err)

	var commitErr *ErrCommitFailed
	assert.ErrorAs(t, err, &commitErr)
	assert.NotEmpty(t, commitErr.Output)
}
