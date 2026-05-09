//go:build !windows

package update

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestTarGz(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, "archive.tar.gz")
	f, err := os.Create(path)
	require.NoError(t, err)
	defer f.Close()

	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	hdr := &tar.Header{
		Name: name,
		Mode: 0755,
		Size: int64(len(content)),
	}
	require.NoError(t, tw.WriteHeader(hdr))
	_, err = tw.Write([]byte(content))
	require.NoError(t, err)

	require.NoError(t, tw.Close())
	require.NoError(t, gw.Close())
	return path
}

func TestReplaceBinary_Success(t *testing.T) {
	dir := t.TempDir()
	binaryContent := "#!/bin/sh\necho hello"
	archivePath := createTestTarGz(t, dir, "scry", binaryContent)

	targetPath := filepath.Join(dir, "scry")
	require.NoError(t, os.WriteFile(targetPath, []byte("old"), 0755))

	err := ReplaceBinary(archivePath, targetPath)
	require.NoError(t, err)

	got, err := os.ReadFile(targetPath)
	require.NoError(t, err)
	assert.Equal(t, binaryContent, string(got))

	info, err := os.Stat(targetPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0755), info.Mode().Perm())
}

func TestReplaceBinary_BinaryNotInArchive(t *testing.T) {
	dir := t.TempDir()
	archivePath := createTestTarGz(t, dir, "other-binary", "content")

	targetPath := filepath.Join(dir, "scry")
	require.NoError(t, os.WriteFile(targetPath, []byte("old"), 0755))

	err := ReplaceBinary(archivePath, targetPath)
	assert.ErrorIs(t, err, ErrReplaceFailed)
}

func TestReplaceBinary_ReadOnlyDirectory(t *testing.T) {
	dir := t.TempDir()
	archivePath := createTestTarGz(t, dir, "scry", "content")

	roDir := filepath.Join(dir, "readonly")
	require.NoError(t, os.MkdirAll(roDir, 0755))
	targetPath := filepath.Join(roDir, "scry")
	require.NoError(t, os.WriteFile(targetPath, []byte("old"), 0755))
	require.NoError(t, os.Chmod(roDir, 0555))
	t.Cleanup(func() { os.Chmod(roDir, 0755) })

	err := ReplaceBinary(archivePath, targetPath)
	assert.ErrorIs(t, err, ErrReplaceFailed)
}

func TestReplaceBinary_CleansUpTempOnFailure(t *testing.T) {
	dir := t.TempDir()
	// Archive with wrong binary name means extraction finds nothing,
	// but let's test with an invalid archive path
	err := ReplaceBinary("/nonexistent.tar.gz", filepath.Join(dir, "scry"))
	assert.ErrorIs(t, err, ErrReplaceFailed)

	// No temp files left behind
	entries, _ := os.ReadDir(dir)
	assert.Empty(t, entries)
}
