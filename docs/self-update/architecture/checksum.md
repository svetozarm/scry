# Module: Checksum Verification (`internal/update/checksum.go`)

## Responsibility

Compute SHA-256 hashes of downloaded files and parse GoReleaser's `checksums.txt` format to verify integrity.

## Interface

```go
// VerifyChecksum computes the SHA-256 of archivePath and compares it against
// the entry for archiveName in checksumsPath. Returns ErrChecksumMismatch on failure.
func VerifyChecksum(archivePath, archiveName, checksumsPath string) error
```

## checksums.txt Format

GoReleaser produces lines in the format:
```
<sha256hex>  <filename>
```

Example:
```
a1b2c3d4...  scry_1.2.3_linux_amd64.tar.gz
e5f6a7b8...  scry_1.2.3_darwin_arm64.tar.gz
```

## Implementation

1. Read `checksumsPath`, split into lines.
2. Find the line where the filename matches `archiveName`.
3. If not found → return `ErrChecksumMismatch` (or a specific "entry not found" variant).
4. Open `archivePath`, compute SHA-256 via `crypto/sha256` + `io.Copy`.
5. Compare hex strings (case-insensitive).
6. Mismatch → return `ErrChecksumMismatch`.

## Testing

- Create temp files with known content, generate expected checksums.
- Test: match, mismatch, missing entry, malformed checksums.txt.

## Relevant Requirements

REQ-E-004, REQ-X-002
