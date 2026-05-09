//go:build windows

package update

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestZip(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, "archive.zip")
	f, err := os.Create(path)
	require.NoError(t, err)
	defer f.Close()

	zw := zip.NewWriter(f)
	w, err := zw.Create(name)
	require.NoError(t, err)
	_, err = w.Write([]byte(content))
	require.NoError(t, err)
	require.NoError(t, zw.Close())
	return path
}

func TestReplaceBinary_Success(t *testing.T) {
	dir := t.TempDir()
	binaryContent := "MZ fake exe content"
	archivePath := createTestZip(t, dir, "scry.exe", binaryContent)

	targetPath := filepath.Join(dir, "scry.exe")
	require.NoError(t, os.WriteFile(targetPath, []byte("old"), 0755))

	err := ReplaceBinary(archivePath, targetPath)
	require.NoError(t, err)

	got, err := os.ReadFile(targetPath)
	require.NoError(t, err)
	assert.Equal(t, binaryContent, string(got))

	// .old file should be cleaned up
	_, err = os.Stat(targetPath + ".old")
	assert.True(t, os.IsNotExist(err))
}

func TestReplaceBinary_BinaryNotInArchive(t *testing.T) {
	dir := t.TempDir()
	archivePath := createTestZip(t, dir, "other.exe", "content")

	targetPath := filepath.Join(dir, "scry.exe")
	require.NoError(t, os.WriteFile(targetPath, []byte("old"), 0755))

	err := ReplaceBinary(archivePath, targetPath)
	assert.ErrorIs(t, err, ErrReplaceFailed)
}

func TestReplaceBinary_ReadOnlyDirectory(t *testing.T) {
	dir := t.TempDir()
	archivePath := createTestZip(t, dir, "scry.exe", "content")

	roDir := filepath.Join(dir, "readonly")
	require.NoError(t, os.MkdirAll(roDir, 0755))
	targetPath := filepath.Join(roDir, "scry.exe")
	require.NoError(t, os.WriteFile(targetPath, []byte("old"), 0755))
	require.NoError(t, os.Chmod(roDir, 0555))
	t.Cleanup(func() { os.Chmod(roDir, 0755) })

	err := ReplaceBinary(archivePath, targetPath)
	assert.ErrorIs(t, err, ErrReplaceFailed)
}

func TestReplaceBinary_CleansUpTempOnFailure(t *testing.T) {
	dir := t.TempDir()
	err := ReplaceBinary("/nonexistent.zip", filepath.Join(dir, "scry.exe"))
	assert.ErrorIs(t, err, ErrReplaceFailed)

	entries, _ := os.ReadDir(dir)
	assert.Empty(t, entries)
}
