package git

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

type ErrCommitFailed struct {
	Output string
}

func (e *ErrCommitFailed) Error() string {
	return fmt.Sprintf("git commit failed: %s", e.Output)
}

var ErrNoRepo = errors.New("not inside a git repository")
var ErrNoStagedChanges = errors.New("no staged changes detected")

func EnsureRepo(dir string) error {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		return ErrNoRepo
	}
	return nil
}

func Diff(dir string) (string, error) {
	cmd := exec.Command("git", "diff", "--cached")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// DiffFileNames returns the list of staged file paths.
func DiffFileNames(dir string) ([]string, error) {
	cmd := exec.Command("git", "diff", "--cached", "--name-only")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	raw := strings.TrimSpace(string(out))
	if raw == "" {
		return nil, nil
	}
	return strings.Split(raw, "\n"), nil
}

// DiffFile returns the staged diff for a single file.
func DiffFile(dir, file string) (string, error) {
	cmd := exec.Command("git", "diff", "--cached", "--", file)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func EnsureStagedChanges(dir string) error {
	cmd := exec.Command("git", "diff", "--cached", "--quiet")
	cmd.Dir = dir
	if err := cmd.Run(); err == nil {
		return ErrNoStagedChanges
	}
	return nil
}

func BranchName(dir string) (string, error) {
	cmd := exec.Command("git", "symbolic-ref", "--short", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func Author(dir string) (string, error) {
	cmd := exec.Command("git", "config", "user.name")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func Commit(dir, message string) (string, error) {
	cmd := exec.Command("git", "commit", "-m", message)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", &ErrCommitFailed{Output: strings.TrimSpace(string(out))}
	}
	return string(out), nil
}
