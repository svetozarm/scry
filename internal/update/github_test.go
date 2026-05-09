package update

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReleaseChecker_InterfaceCompliance(t *testing.T) {
	// Verify GitHubClient implements ReleaseChecker
	var _ ReleaseChecker = (*GitHubClient)(nil)
}

func TestNewGitHubClient(t *testing.T) {
	client := NewGitHubClient("owner/repo")
	assert.NotNil(t, client)
	assert.NotNil(t, client.httpClient)
	assert.Equal(t, "owner/repo", client.repo)
}

func TestRelease_HasExpectedFields(t *testing.T) {
	r := Release{
		Tag: "v1.2.3",
		Assets: []Asset{
			{Name: "scry_1.2.3_linux_amd64.tar.gz", URL: "https://example.com/asset"},
		},
	}
	assert.Equal(t, "v1.2.3", r.Tag)
	assert.Len(t, r.Assets, 1)
	assert.Equal(t, "scry_1.2.3_linux_amd64.tar.gz", r.Assets[0].Name)
	assert.Equal(t, "https://example.com/asset", r.Assets[0].URL)
}

// Stub methods to satisfy interface — will be properly implemented in 2.2/2.3
func TestGitHubClient_LatestRelease_NotImplementedYet(t *testing.T) {
	client := NewGitHubClient("svetozarm/scry")
	_, err := client.LatestRelease(context.Background())
	// For now just verify it doesn't panic; real tests come in 2.2
	_ = err
}

func TestGitHubClient_DownloadAsset_NotImplementedYet(t *testing.T) {
	client := NewGitHubClient("svetozarm/scry")
	err := client.DownloadAsset(context.Background(), "http://example.com", "/tmp/test")
	// For now just verify it doesn't panic; real tests come in 2.3
	_ = err
}
