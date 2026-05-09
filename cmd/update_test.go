package cmd

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/svetozarm/scry/internal/update"
)

type mockReleaseChecker struct {
	release    *update.Release
	err        error
	downloadFn func(ctx context.Context, url string, dest string) error
}

func (m *mockReleaseChecker) LatestRelease(_ context.Context) (*update.Release, error) {
	return m.release, m.err
}

func (m *mockReleaseChecker) DownloadAsset(_ context.Context, _ string, dest string) error {
	if m.downloadFn != nil {
		return m.downloadFn(context.Background(), "", dest)
	}
	return nil
}

func setupUpdateTest(t *testing.T, version string, checker update.ReleaseChecker) *bytes.Buffer {
	t.Helper()
	origVersion := Version
	origChecker := newChecker
	origNI := nonInteractive
	t.Cleanup(func() {
		Version = origVersion
		newChecker = origChecker
		nonInteractive = origNI
	})

	Version = version
	newChecker = func() update.ReleaseChecker { return checker }
	nonInteractive = true

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetArgs([]string{"update", "--non-interactive"})
	return &stdout
}

func TestUpdate_NonInteractive_DevBuild(t *testing.T) {
	stdout := setupUpdateTest(t, "", &mockReleaseChecker{})

	err := rootCmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "Warning: update unavailable for development builds\n", stdout.String())
}

func TestUpdate_NonInteractive_UpToDate(t *testing.T) {
	checker := &mockReleaseChecker{
		release: &update.Release{Tag: "v1.2.0"},
	}
	stdout := setupUpdateTest(t, "v1.2.0", checker)

	err := rootCmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "Already up to date: v1.2.0\n", stdout.String())
}

func TestUpdate_NonInteractive_APIError(t *testing.T) {
	checker := &mockReleaseChecker{err: update.ErrUpdateAPI}
	setupUpdateTest(t, "v1.0.0", checker)

	r, w, err := os.Pipe()
	require.NoError(t, err)
	origStderr := os.Stderr
	os.Stderr = w

	execErr := rootCmd.Execute()

	w.Close()
	os.Stderr = origStderr
	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	stderr := string(buf[:n])

	assertSilentError(t, execErr, 5)
	assert.Contains(t, stderr, "update failed: could not reach GitHub")
}
