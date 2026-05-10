package update

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
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

func TestRun_AssetNotFound(t *testing.T) {
	checker := &mockChecker{
		release: &Release{Tag: "v2.0.0", Assets: []Asset{
			{Name: "unrelated_file.tar.gz", URL: "http://example.com/unrelated"},
		}},
	}
	_, err := Run(context.Background(), "v1.0.0", checker)
	require.ErrorIs(t, err, ErrAssetNotFound)
}

func TestRun_PermissionDenied(t *testing.T) {
	assetName := deriveAssetName("v2.0.0")
	checker := &mockChecker{
		release: &Release{Tag: "v2.0.0", Assets: []Asset{
			{Name: assetName, URL: "http://example.com/archive"},
			{Name: "checksums.txt", URL: "http://example.com/checksums"},
			{Name: "checksums.txt.sig", URL: "http://example.com/checksums.sig"},
		}},
	}
	// Use a non-writable path to trigger permission error
	origExec := executablePath
	readonlyDir := t.TempDir()
	os.Chmod(readonlyDir, 0555)
	t.Cleanup(func() {
		os.Chmod(readonlyDir, 0755)
		executablePath = origExec
	})
	executablePath = func() (string, error) {
		return filepath.Join(readonlyDir, "scry"), nil
	}

	_, err := Run(context.Background(), "v1.0.0", checker)
	require.ErrorIs(t, err, ErrPermission)
}

func TestRun_FullPipeline(t *testing.T) {
	dir := t.TempDir()

	// Generate test key pair and override embedded key
	privKey, pubPEM := generateTestKeyPair(t)
	origKey := cosignPubKey
	cosignPubKey = pubPEM
	t.Cleanup(func() { cosignPubKey = origKey })

	// Create a fake binary to replace
	binaryPath := filepath.Join(dir, "scry")
	os.WriteFile(binaryPath, []byte("old binary"), 0755)

	// Create a valid tar.gz archive containing "scry"
	assetName := deriveAssetName("v2.0.0")
	archivePath := filepath.Join(dir, assetName)
	createTestArchive(t, archivePath, "scry", []byte("new binary"))

	// Compute checksum of the archive
	archiveContent, _ := os.ReadFile(archivePath)
	hash := sha256.Sum256(archiveContent)
	hashHex := hex.EncodeToString(hash[:])
	checksumsContent := fmt.Sprintf("%s  %s\n", hashHex, assetName)

	// Sign the checksums
	checksumsDigest := sha256.Sum256([]byte(checksumsContent))
	sigBytes, err := ecdsa.SignASN1(rand.Reader, privKey, checksumsDigest[:])
	require.NoError(t, err)
	sigB64 := base64.StdEncoding.EncodeToString(sigBytes)

	checker := &mockChecker{
		release: &Release{Tag: "v2.0.0", Assets: []Asset{
			{Name: assetName, URL: "http://example.com/archive"},
			{Name: "checksums.txt", URL: "http://example.com/checksums"},
			{Name: "checksums.txt.sig", URL: "http://example.com/checksums.sig"},
		}},
		downloadFn: func(ctx context.Context, url string, dest string) error {
			switch filepath.Base(dest) {
			case assetName:
				return os.WriteFile(dest, archiveContent, 0644)
			case "checksums.txt":
				return os.WriteFile(dest, []byte(checksumsContent), 0644)
			case "checksums.txt.sig":
				return os.WriteFile(dest, []byte(sigB64), 0644)
			}
			return nil
		},
	}

	origExec := executablePath
	t.Cleanup(func() { executablePath = origExec })
	executablePath = func() (string, error) { return binaryPath, nil }

	result, err := Run(context.Background(), "v1.0.0", checker)
	require.NoError(t, err)
	assert.Equal(t, "v1.0.0", result.OldVersion)
	assert.Equal(t, "v2.0.0", result.NewVersion)

	// Verify binary was replaced
	content, _ := os.ReadFile(binaryPath)
	assert.Equal(t, "new binary", string(content))

	// Verify temp files cleaned up
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		assert.Equal(t, "scry", e.Name(), "unexpected leftover file: %s", e.Name())
	}
}

func createTestArchive(t *testing.T, archivePath, name string, content []byte) {
	t.Helper()
	f, err := os.Create(archivePath)
	require.NoError(t, err)
	defer f.Close()

	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)
	tw.WriteHeader(&tar.Header{Name: name, Size: int64(len(content)), Mode: 0755, Typeflag: tar.TypeReg})
	tw.Write(content)
	tw.Close()
	gw.Close()
}

type mockChecker struct {
	release    *Release
	err        error
	downloadFn func(ctx context.Context, url string, dest string) error
}

func (m *mockChecker) LatestRelease(ctx context.Context) (*Release, error) {
	return m.release, m.err
}

func (m *mockChecker) DownloadAsset(ctx context.Context, url string, dest string) error {
	if m.downloadFn != nil {
		return m.downloadFn(ctx, url, dest)
	}
	return nil
}
