# Module: Self-Update (`internal/update/`)

## Overview

The self-update module adds a `scry update` command that checks GitHub Releases for a newer version, downloads the platform-appropriate archive, verifies its SHA-256 checksum, and replaces the running binary in-place.

## System Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                   CLI: `scry update` command                     │
│  Reads embedded version, delegates to update module              │
│  Handles non-interactive mode, error display, exit codes         │
└──────────────┬──────────────────────────────────────────────────┘
               │
               ▼
┌──────────────────────────────────────────────────────────────────┐
│                    Update Orchestrator                            │
│  internal/update/update.go                                       │
│  Coordinates: version check → download → verify → replace       │
└──────┬──────────────┬────────────────┬──────────────┬───────────┘
       │              │                │              │
       ▼              ▼                ▼              ▼
┌────────────┐ ┌────────────┐ ┌──────────────┐ ┌──────────────┐
│  GitHub    │ │  Download  │ │  Checksum    │ │  Replace     │
│  Client    │ │            │ │  Verify      │ │  Binary      │
│            │ │  Fetch     │ │              │ │              │
│  Latest    │ │  archive + │ │  SHA-256     │ │  Extract +   │
│  release   │ │  checksums │ │  compare     │ │  atomic swap │
│  tag       │ │  to tmpdir │ │              │ │  (or rename) │
└────────────┘ └────────────┘ └──────────────┘ └──────────────┘
```

## Package Structure

```
internal/update/
  update.go          # Updater struct, Run() orchestrator
  github.go          # GitHub Releases API client (latest release, asset URLs)
  download.go        # HTTP download with temp file management
  checksum.go        # SHA-256 computation and checksums.txt parsing
  replace.go         # Binary extraction and replacement (platform-aware)
  replace_windows.go # Windows-specific rename-and-replace strategy
  errors.go          # Typed errors for the update module
  update_test.go     # Unit tests (httptest-based)
```

## Data Flow

1. CLI passes embedded version string to `update.Run()`
2. **GitHub client** queries `GET /repos/svetozarm/scry/releases/latest` → gets tag + asset list
3. **Version compare**: if tag == embedded version → "already up to date", exit 0
4. **Asset resolution**: derive expected filename from `runtime.GOOS`/`runtime.GOARCH` → find in asset list
5. **Download**: fetch archive + `checksums.txt` to temp files in same directory as binary
6. **Checksum verify**: compute SHA-256 of archive, parse checksums.txt, compare
7. **Replace**: extract binary from archive, swap with current executable
8. **Cleanup**: remove temp files, report success

## Key Design Decisions

### No External Dependencies
The module uses only Go stdlib (`net/http`, `crypto/sha256`, `archive/tar`, `compress/gzip`, `archive/zip`, `os`, `runtime`). No GitHub SDK.

### Interface for Testability
The GitHub HTTP calls are behind an interface so tests can use `httptest.Server`:

```go
type ReleaseChecker interface {
    LatestRelease(ctx context.Context) (*Release, error)
    DownloadAsset(ctx context.Context, url string, dest string) error
}
```

### Platform-Aware Binary Replacement
- **Unix**: Write new binary to temp path in same dir → `os.Rename()` (atomic on same filesystem) → `os.Chmod(0755)`
- **Windows**: Rename running binary → write new binary to original path → delete old. Uses build tag `replace_windows.go` with `//go:build windows`.

### Version Comparison
Simple semver string comparison after stripping the `v` prefix. Uses `>=` logic — if remote tag is not strictly newer, report "up to date". No need for a semver library; split on `.` and compare integers.

### Error Types

```go
var (
    ErrUpdateAPI         // GitHub API unreachable or non-200 (exit code 5)
    ErrChecksumMismatch  // SHA-256 mismatch (exit code 5)
    ErrAssetNotFound     // Platform asset missing from release (exit code 5)
    ErrPermission        // Binary path not writable (exit code 5)
    ErrReplaceFailed     // Rename-and-replace failed (exit code 5)
    ErrDevBuild          // No version string embedded (exit code 0, warning only)
)
```

### Exit Code
All update errors use exit code **5** (distinct from git=2, provider=3, config=4).

## Integration with Existing CLI

The `cmd/` layer gets a new file `cmd/update.go` that:
- Registers `updateCmd` as a subcommand on `rootCmd`
- Passes `version` (from `main.go`) and `nonInteractive` flag to the update module
- Maps update errors to styled output via existing `handleError` pattern

## Dependencies

- Go stdlib only: `net/http`, `crypto/sha256`, `archive/tar`, `compress/gzip`, `archive/zip`, `encoding/json`, `os`, `runtime`, `io`, `path/filepath`

## Relevant Requirements

REQ-U-001, REQ-U-002, REQ-U-003, REQ-E-001–007, REQ-S-001, REQ-S-002, REQ-X-001–007, REQ-NF-001–006
