package update

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const defaultBaseURL = "https://api.github.com"

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
	baseURL    string
}

// NewGitHubClient creates a GitHubClient for the given "owner/repo".
func NewGitHubClient(repo string) *GitHubClient {
	return &GitHubClient{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		repo:       repo,
		baseURL:    defaultBaseURL,
	}
}

func (c *GitHubClient) LatestRelease(ctx context.Context) (*Release, error) {
	url := fmt.Sprintf("%s/repos/%s/releases/latest", c.baseURL, c.repo)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUpdateAPI, err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUpdateAPI, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: HTTP %d", ErrUpdateAPI, resp.StatusCode)
	}

	var ghRelease struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&ghRelease); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUpdateAPI, err)
	}

	rel := &Release{Tag: ghRelease.TagName}
	for _, a := range ghRelease.Assets {
		rel.Assets = append(rel.Assets, Asset{Name: a.Name, URL: a.BrowserDownloadURL})
	}
	return rel, nil
}

func (c *GitHubClient) DownloadAsset(ctx context.Context, url string, dest string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUpdateAPI, err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUpdateAPI, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: HTTP %d", ErrUpdateAPI, resp.StatusCode)
	}

	dir := filepath.Dir(dest)
	tmp, err := os.CreateTemp(dir, ".scry-download-*")
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUpdateAPI, err)
	}
	tmpPath := tmp.Name()

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("%w: %v", ErrUpdateAPI, err)
	}

	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("%w: %v", ErrUpdateAPI, err)
	}

	if err := os.Rename(tmpPath, dest); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("%w: %v", ErrUpdateAPI, err)
	}

	return nil
}
