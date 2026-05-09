# Module: Binary Replacement (`internal/update/replace.go`)

## Responsibility

Extract the Scry binary from the downloaded archive and replace the currently running executable. Platform-aware: uses atomic rename on Unix and rename-and-replace on Windows.

## Interface

```go
// ReplaceBinary extracts the binary from archivePath and replaces the
// executable at binaryPath. Handles platform-specific locking issues.
func ReplaceBinary(archivePath, binaryPath string) error
```

## Unix Strategy (`replace.go`)

1. Open archive (tar.gz), find the `scry` entry.
2. Write extracted binary to a temp file in the same directory as `binaryPath`.
3. `os.Chmod(tempPath, 0755)` — set executable bits.
4. `os.Rename(tempPath, binaryPath)` — atomic swap (same filesystem guaranteed by download strategy).
5. On failure: remove temp file, return `ErrReplaceFailed`.

## Windows Strategy (`replace_windows.go`, `//go:build windows`)

On Windows, the running executable cannot be overwritten or deleted. However, it **can** be renamed.

1. Open archive (zip), find the `scry.exe` entry.
2. Write extracted binary to a temp file in the same directory.
3. Rename current binary: `binaryPath` → `binaryPath + ".old"`.
4. Rename new binary: `tempPath` → `binaryPath`.
5. Attempt to delete `binaryPath + ".old"` (best-effort; may fail if still locked).
6. If step 3 or 4 fails → return `ErrReplaceFailed` with suggestion for manual replacement.

## Archive Format Detection

- `.tar.gz` → `archive/tar` + `compress/gzip`
- `.zip` → `archive/zip`

Determined by file extension of the archive path.

## Testing

- Create test archives (tar.gz and zip) containing a known binary.
- Verify extraction produces correct content.
- Verify permissions on Unix.
- Test failure when target directory is read-only.

## Relevant Requirements

REQ-E-005, REQ-X-004, REQ-X-005, REQ-NF-004
