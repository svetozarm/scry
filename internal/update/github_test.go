package update

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReleaseChecker_InterfaceCompliance(t *testing.T) {
	var _ ReleaseChecker = (*GitHubClient)(nil)
}

func TestNewGitHubClient(t *testing.T) {
	client := NewGitHubClient("owner/repo")
	assert.NotNil(t, client)
	assert.NotNil(t, client.httpClient)
	assert.Equal(t, "owner/repo", client.repo)
}

func TestLatestRelease_Success(t *testing.T) {
	body := `{
		"tag_name": "v1.4.0",
		"assets": [
			{"name": "scry_1.4.0_linux_amd64.tar.gz", "browser_download_url": "https://example.com/linux.tar.gz"},
			{"name": "scry_1.4.0_darwin_arm64.tar.gz", "browser_download_url": "https://example.com/darwin.tar.gz"}
		]
	}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/repos/owner/repo/releases/latest", r.URL.Path)
		assert.Equal(t, "application/vnd.github+json", r.Header.Get("Accept"))
		fmt.Fprint(w, body)
	}))
	defer srv.Close()

	client := NewGitHubClient("owner/repo")
	client.baseURL = srv.URL

	rel, err := client.LatestRelease(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "v1.4.0", rel.Tag)
	assert.Len(t, rel.Assets, 2)
	assert.Equal(t, "scry_1.4.0_linux_amd64.tar.gz", rel.Assets[0].Name)
	assert.Equal(t, "https://example.com/linux.tar.gz", rel.Assets[0].URL)
}

func TestLatestRelease_Non200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	client := NewGitHubClient("owner/repo")
	client.baseURL = srv.URL

	_, err := client.LatestRelease(context.Background())
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUpdateAPI))
}

func TestLatestRelease_RateLimited(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	client := NewGitHubClient("owner/repo")
	client.baseURL = srv.URL

	_, err := client.LatestRelease(context.Background())
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUpdateAPI))
	assert.Contains(t, err.Error(), "403")
}

func TestLatestRelease_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "not json at all{{{")
	}))
	defer srv.Close()

	client := NewGitHubClient("owner/repo")
	client.baseURL = srv.URL

	_, err := client.LatestRelease(context.Background())
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUpdateAPI))
}

func TestLatestRelease_CancelledContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"tag_name":"v1.0.0","assets":[]}`)
	}))
	defer srv.Close()

	client := NewGitHubClient("owner/repo")
	client.baseURL = srv.URL

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.LatestRelease(ctx)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUpdateAPI))
}

func TestDownloadAsset_Success(t *testing.T) {
	content := []byte("binary content here 1234567890")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(content)
	}))
	defer srv.Close()

	client := NewGitHubClient("owner/repo")
	dest := filepath.Join(t.TempDir(), "downloaded_file")

	err := client.DownloadAsset(context.Background(), srv.URL+"/asset.tar.gz", dest)
	require.NoError(t, err)

	got, err := os.ReadFile(dest)
	require.NoError(t, err)
	assert.Equal(t, content, got)
}

func TestDownloadAsset_Non200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := NewGitHubClient("owner/repo")
	dest := filepath.Join(t.TempDir(), "downloaded_file")

	err := client.DownloadAsset(context.Background(), srv.URL+"/missing", dest)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUpdateAPI))

	_, statErr := os.Stat(dest)
	assert.True(t, os.IsNotExist(statErr))
}

func TestDownloadAsset_CancelledContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("data"))
	}))
	defer srv.Close()

	client := NewGitHubClient("owner/repo")
	dir := t.TempDir()
	dest := filepath.Join(dir, "downloaded_file")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := client.DownloadAsset(ctx, srv.URL+"/asset", dest)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUpdateAPI))

	// Verify no temp files left behind
	entries, _ := os.ReadDir(dir)
	assert.Empty(t, entries)
}

func TestDownloadAsset_InterruptedMidTransfer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		// Cancel context while transfer is in progress
		cancel()
	}))
	defer srv.Close()

	client := NewGitHubClient("owner/repo")
	dir := t.TempDir()
	dest := filepath.Join(dir, "downloaded_file")

	err := client.DownloadAsset(ctx, srv.URL+"/asset", dest)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUpdateAPI))

	// Verify no temp files left behind
	entries, _ := os.ReadDir(dir)
	assert.Empty(t, entries)
}
