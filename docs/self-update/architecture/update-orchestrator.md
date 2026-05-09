# Module: Update Orchestrator (`internal/update/update.go`)

## Responsibility

Top-level coordination of the update workflow. Accepts the embedded version, determines if an update is needed, and drives the download → verify → replace pipeline.

## Interface

```go
type Result struct {
    OldVersion string
    NewVersion string
    UpToDate   bool
}

// Run executes the full update workflow. Returns a Result on success.
func Run(ctx context.Context, currentVersion string, checker ReleaseChecker) (*Result, error)
```

## Orchestration Logic

```
func Run(ctx, currentVersion, checker):
    if currentVersion == "":
        return ErrDevBuild

    release := checker.LatestRelease(ctx)
    if !isNewer(release.Tag, currentVersion):
        return Result{UpToDate: true}

    assetName := deriveAssetName(release.Tag)
    archiveAsset := findAsset(release.Assets, assetName)
    checksumAsset := findAsset(release.Assets, "checksums.txt")
    if archiveAsset == nil || checksumAsset == nil:
        return ErrAssetNotFound

    binaryPath := os.Executable()
    dir := filepath.Dir(binaryPath)

    // Check write permission early
    if !writable(dir):
        return ErrPermission

    archiveDest := filepath.Join(dir, assetName)
    checksumDest := filepath.Join(dir, "checksums.txt")

    checker.DownloadAsset(ctx, archiveAsset.URL, archiveDest)
    checker.DownloadAsset(ctx, checksumAsset.URL, checksumDest)

    defer cleanup(archiveDest, checksumDest)

    VerifyChecksum(archiveDest, assetName, checksumDest)
    ReplaceBinary(archiveDest, binaryPath)

    return Result{OldVersion: currentVersion, NewVersion: release.Tag}
```

## Version Comparison

```go
// isNewer returns true if remote is strictly newer than current.
// Strips "v" prefix, splits on ".", compares integers left-to-right.
func isNewer(remote, current string) bool
```

## Asset Name Derivation

```go
// deriveAssetName builds the expected filename from version + runtime.GOOS/GOARCH.
// Pattern: scry_<version>_<os>_<arch>.<ext>
// Extension: .zip for windows, .tar.gz otherwise.
func deriveAssetName(tag string) string
```

## Testing

- Mock `ReleaseChecker` interface.
- Test: up-to-date, newer version available, dev build, asset not found, permission denied.
- Integration test with `httptest.Server` for full flow.

## Relevant Requirements

REQ-E-001 through REQ-E-007, REQ-S-002, REQ-X-006
