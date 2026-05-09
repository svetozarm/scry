package update

import (
	"context"
	"net/http"
	"time"
)

// Release represents a GitHub release with its tag and downloadable assets.
type Release struct {
	Tag    string
	Assets []Asset
}

// Asset represents a downloadable file attached to a release.
type Asset struct {
	Name string
	URL  string
}

// ReleaseChecker abstracts GitHub release operations for testability.
type ReleaseChecker interface {
	LatestRelease(ctx context.Context) (*Release, error)
	DownloadAsset(ctx context.Context, url string, dest string) error
}

// GitHubClient implements ReleaseChecker using the GitHub Releases API.
type GitHubClient struct {
	httpClient *http.Client
	repo       string
}

// NewGitHubClient creates a GitHubClient for the given "owner/repo".
func NewGitHubClient(repo string) *GitHubClient {
	return &GitHubClient{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		repo:       repo,
	}
}

func (c *GitHubClient) LatestRelease(ctx context.Context) (*Release, error) {
	// TODO: implement in task 2.2
	return nil, nil
}

func (c *GitHubClient) DownloadAsset(ctx context.Context, url string, dest string) error {
	// TODO: implement in task 2.3
	return nil
}
