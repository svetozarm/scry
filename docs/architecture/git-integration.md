# Module: Git Integration (`internal/git/`)

## Responsibility

All interactions with the local Git repository: detecting the repo, checking for staged changes, retrieving the diff, reading branch/author metadata, and executing commits.

## Package Structure

```
internal/git/
  git.go       # Public API: EnsureRepo, EnsureStagedChanges, Diff, BranchName, Author, Commit
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

// BranchName returns the current branch name.
func BranchName(dir string) (string, error)

// Author returns the configured git author (user.name).
func Author(dir string) (string, error)

// Commit executes `git commit -m <message>` and returns the combined output.
func Commit(dir, message string) (string, error)
```

## Implementation Notes

- All functions shell out to `git` via `os/exec`. Git is assumed to be on `$PATH` (per PRD assumptions).
- `EnsureRepo` runs `git rev-parse --is-inside-work-tree`.
- `EnsureStagedChanges` runs `git diff --cached --quiet` and checks the exit code (1 = changes exist).
- `BranchName` runs `git symbolic-ref --short HEAD`.
- `Author` runs `git config user.name`.
- Commands use `exec.Command` with `cmd.Dir` set to the provided directory.

## Error Types

- `ErrNoRepo` — not inside a git repository (REQ-S-002)
- `ErrNoStagedChanges` — no staged changes detected (REQ-S-003)
- `ErrCommitFailed{Output}` — git commit returned non-zero (REQ-X-007)

## Testing Strategy

Unit tests create temporary git repos with `git init`, stage files, and verify each function. No mocking of git — these are integration-style tests against real git.

## Dependencies

- `os/exec`

## Relevant Requirements

REQ-E-001, REQ-E-004, REQ-S-002, REQ-S-003, REQ-X-007
