# Module: Download (`internal/update/download.go`)

## Responsibility

Download release assets (archive + checksums.txt) to temporary files in the same directory as the running binary. Handle network failures gracefully with cleanup.

## Implementation

```go
// DownloadAsset fetches url and writes it to dest. Returns error on failure.
// On network interruption, partial files are removed.
func (c *GitHubClient) DownloadAsset(ctx context.Context, url string, dest string) error
```

### Strategy

1. Resolve binary directory via `os.Executable()` + `filepath.Dir()`.
2. Create temp file in that directory (`os.CreateTemp(dir, "scry-update-*")`).
3. HTTP GET with context (inherits timeout/cancellation).
4. Stream response body to temp file via `io.Copy`.
5. On success: rename temp file to final destination name.
6. On failure: remove temp file, return wrapped error.

### Why Same Directory?

`os.Rename()` is atomic only within the same filesystem. Downloading to the binary's directory guarantees the final rename (during replacement) is atomic on Unix.

## Error Handling

- Context cancelled / deadline exceeded → clean up temp file, return `ErrUpdateAPI` wrapping "download interrupted"
- HTTP non-200 → clean up temp file, return `ErrUpdateAPI` with status
- Write error (disk full) → clean up temp file, return wrapped OS error

## Testing

- `httptest.Server` serves a known payload.
- Verify file content matches.
- Verify cleanup on cancelled context.

## Relevant Requirements

REQ-E-002, REQ-E-003, REQ-X-007, REQ-NF-002
