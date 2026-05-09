package update

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun_DevBuild(t *testing.T) {
	_, err := Run(context.Background(), "", nil)
	require.ErrorIs(t, err, ErrDevBuild)
}

func TestRun_UpToDate(t *testing.T) {
	checker := &mockChecker{
		release: &Release{Tag: "v1.0.0", Assets: []Asset{}},
	}
	result, err := Run(context.Background(), "v1.0.0", checker)
	require.NoError(t, err)
	assert.True(t, result.UpToDate)
}

func TestRun_APIError(t *testing.T) {
	checker := &mockChecker{err: ErrUpdateAPI}
	_, err := Run(context.Background(), "v1.0.0", checker)
	require.ErrorIs(t, err, ErrUpdateAPI)
}

type mockChecker struct {
	release *Release
	err     error
}

func (m *mockChecker) LatestRelease(ctx context.Context) (*Release, error) {
	return m.release, m.err
}

func (m *mockChecker) DownloadAsset(ctx context.Context, url string, dest string) error {
	return nil
}
