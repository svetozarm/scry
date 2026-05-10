# Module: Git Integration (`internal/git/`)

## Responsibility

All interactions with the local Git repository: detecting the repo, checking for staged changes, retrieving the diff (full and per-file), reading branch/author metadata, and executing commits.

## Package Structure

```
internal/git/
  git.go       # Public API: EnsureRepo, EnsureStagedChanges, Diff, DiffFileNames, DiffFile, BranchName, Author, Commit
  git_test.go  # Unit tests (using temp repos)
```

## Public API

All functions accept a `dir string` parameter specifying the working directory for git commands.

```go
// EnsureRepo checks that dir is inside a git repo. Returns ErrNoRepo if not.
func EnsureRepo(dir string) error

// EnsureStagedChanges checks that there are staged changes. Returns ErrNoStagedChanges if not.
func EnsureStagedChanges(dir string) error

// Diff returns the output of `git diff --cached`.
func Diff(dir string) (string, error)

// DiffFileNames returns the list of staged file paths.
func DiffFileNames(dir string) ([]string, error)

// DiffFile returns the staged diff for a single file.
func DiffFile(dir, file string) (string, error)

// BranchName returns the current branch name.
func BranchName(dir string) (string, error)

// Author returns the configured git author (user.name).
func Author(dir string) (string, error)

// Commit executes `git commit -m <message>` and returns the combined output.
func Commit(dir, message string) (string, error)
```

## Implementation Notes

- All functions shell out to `git` via `os/exec`. Git is assumed to be on `$PATH`.
- `EnsureRepo` runs `git rev-parse --is-inside-work-tree`.
- `EnsureStagedChanges` runs `git diff --cached --quiet` and checks the exit code (1 = changes exist).
- `DiffFileNames` runs `git diff --cached --name-only` and splits on newlines.
- `DiffFile` runs `git diff --cached -- <file>` for a single file's staged diff.
- `BranchName` runs `git symbolic-ref --short HEAD`.
- `Author` runs `git config user.name`.
- Commands use `exec.Command` with `cmd.Dir` set to the provided directory.

## Error Types

- `ErrNoRepo` — not inside a git repository
- `ErrNoStagedChanges` — no staged changes detected
- `ErrCommitFailed{Output}` — git commit returned non-zero

## Testing Strategy

Unit tests create temporary git repos with `git init`, stage files, and verify each function. No mocking of git — these are integration-style tests against real git.

## Dependencies

- `os/exec`
- `strings`
