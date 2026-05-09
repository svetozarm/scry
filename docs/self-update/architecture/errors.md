# Module: Update Errors (`internal/update/errors.go`)

## Responsibility

Define typed sentinel errors for the update module. These are used by the CLI layer to map failures to user-facing messages and exit code 5.

## Error Definitions

```go
var (
    ErrUpdateAPI        = errors.New("update: GitHub API error")
    ErrChecksumMismatch = errors.New("update: checksum mismatch")
    ErrAssetNotFound    = errors.New("update: platform asset not found in release")
    ErrPermission       = errors.New("update: binary path not writable")
    ErrReplaceFailed    = errors.New("update: failed to replace binary")
    ErrDevBuild         = errors.New("update: no version embedded (development build)")
)
```

## Usage Pattern

Each error is wrapped with context using `fmt.Errorf("...: %w", ErrUpdateAPI)` so the CLI layer can match via `errors.Is()` while still carrying descriptive messages.

## CLI Mapping

All update errors map to exit code **5** except `ErrDevBuild` which exits 0 (warning only).

## Relevant Requirements

REQ-X-001 through REQ-X-007
