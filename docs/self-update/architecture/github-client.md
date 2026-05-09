# Module: GitHub Client (`internal/update/github.go`)

## Responsibility

Query the GitHub Releases API for the latest release of `svetozarm/scry`. Parse the response to extract the tag name and asset download URLs.

## Interface

```go
type Release struct {
    Tag    string  // e.g., "v1.2.3"
    Assets []Asset
}

type Asset struct {
    Name string // e.g., "scry_1.2.3_linux_amd64.tar.gz"
    URL  string // browser_download_url
}

type ReleaseChecker interface {
    LatestRelease(ctx context.Context) (*Release, error)
    DownloadAsset(ctx context.Context, url string, dest string) error
}
```

## Implementation

- `GET https://api.github.com/repos/svetozarm/scry/releases/latest`
- No `Authorization` header (public repo, unauthenticated).
- Sets `Accept: application/vnd.github+json` header.
- Timeout: 10 seconds via `context.WithTimeout`.
- Parses only `tag_name` and `assets[].{name, browser_download_url}` from the JSON response.

## Error Mapping

| HTTP Status | Error |
|---|---|
| Network error / timeout | `ErrUpdateAPI` wrapping the underlying error |
| 403 (rate limit) | `ErrUpdateAPI` with "rate limited" message |
| 404 | `ErrUpdateAPI` with "repository not found" message |
| Any non-200 | `ErrUpdateAPI` with status code in message |

## Testing

- `httptest.Server` returns canned JSON responses.
- Tests cover: success, non-200, malformed JSON, timeout (context cancelled).

## Relevant Requirements

REQ-U-003, REQ-E-001, REQ-X-001, REQ-NF-001, REQ-NF-006
