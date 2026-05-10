# Module: Self-Update (`internal/update/`)

## Responsibility

Check for new releases on GitHub, download platform-specific binaries, verify integrity (SHA-256 checksum) and authenticity (cosign signature), and atomically replace the running binary.

## Package Structure

```
internal/update/
  update.go               # Run() — top-level update workflow
  github.go               # GitHubClient, ReleaseChecker interface, download logic
  version.go              # Version comparison (isNewer)
  asset_name.go           # Platform-specific asset name derivation
  checksum.go             # SHA-256 checksum verification
  signature.go            # Cosign signature verification
  archive.go              # Archive extraction (tar.gz)
  replace.go              # Atomic binary replacement (Unix)
  replace_windows.go      # Atomic binary replacement (Windows)
  writable.go             # Directory writability check
  errors.go               # Typed error definitions
  cosign.pub              # Embedded cosign public key
  *_test.go               # Unit tests
```

## Public API

```go
// Result holds the outcome of an update check or execution.
type Result struct {
    OldVersion string
    NewVersion string
    UpToDate   bool
}

// Run executes the full update workflow.
func Run(ctx context.Context, currentVersion string, checker ReleaseChecker) (*Result, error)

// ReleaseChecker abstracts GitHub release operations for testability.
type ReleaseChecker interface {
    LatestRelease(ctx context.Context) (*Release, error)
    DownloadAsset(ctx context.Context, url string, dest string) error
}

// NewGitHubClient creates a GitHubClient for the given "owner/repo".
func NewGitHubClient(repo string) *GitHubClient
```

## Update Workflow

1. Check if `currentVersion` is set (empty = dev build → return `ErrDevBuild`)
2. Query GitHub Releases API for the latest release tag
3. Compare versions; if not newer, return `Result{UpToDate: true}`
4. Derive platform-specific asset name (e.g., `scry_v1.2.0_linux_amd64.tar.gz`)
5. Locate asset URL, checksums URL, and signature URL in release assets
6. Resolve current binary path and verify the directory is writable
7. Download archive, `checksums.txt`, and `checksums.txt.sig`
8. Verify cosign signature on `checksums.txt`
9. Verify SHA-256 checksum of the archive against `checksums.txt`
10. Extract and atomically replace the binary
11. Clean up temporary files

## Security

- **Checksum verification**: SHA-256 hash of the downloaded archive is compared against the signed checksums file.
- **Signature verification**: The `checksums.txt` file is verified against `checksums.txt.sig` using an embedded cosign public key.
- **Download restrictions**: Only HTTPS downloads from `github.com` and `objects.githubusercontent.com` are allowed.
- **Size limit**: Downloads are capped at 50 MB to prevent resource exhaustion.
- **Atomic replacement**: Binary is replaced atomically to prevent corruption on failure.

## Error Types

```go
var (
    ErrUpdateAPI        // GitHub API error (network, HTTP status)
    ErrChecksumMismatch // Downloaded file hash doesn't match
    ErrSignatureInvalid // Cosign signature verification failed
    ErrAssetNotFound    // No release asset for current OS/arch
    ErrPermission       // Binary path not writable
    ErrReplaceFailed    // Could not replace binary
    ErrDevBuild         // No version embedded (development build)
)
```

## Platform Support

Asset names are derived from `runtime.GOOS` and `runtime.GOARCH`. The binary replacement logic has platform-specific implementations:
- **Unix** (`replace.go`): Extract from tar.gz, rename atomically
- **Windows** (`replace_windows.go`): Handles Windows file locking constraints

## Dependencies

- `net/http`
- `crypto/sha256`
- `os`, `path/filepath`
- `runtime` (GOOS, GOARCH)
- `encoding/json`
- Embedded cosign public key (`cosign.pub`)
