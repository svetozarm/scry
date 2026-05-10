package update

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestVerifyChecksum_Match(t *testing.T) {
	dir := t.TempDir()

	archivePath := filepath.Join(dir, "scry_linux_amd64.tar.gz")
	content := []byte("fake archive content")
	os.WriteFile(archivePath, content, 0644)

	hash := sha256.Sum256(content)
	hashHex := hex.EncodeToString(hash[:])

	checksumsPath := filepath.Join(dir, "checksums.txt")
	line := fmt.Sprintf("%s  scry_linux_amd64.tar.gz\n", hashHex)
	os.WriteFile(checksumsPath, []byte(line), 0644)

	err := VerifyChecksum(archivePath, "scry_linux_amd64.tar.gz", checksumsPath)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestVerifyChecksum_Mismatch(t *testing.T) {
	dir := t.TempDir()

	archivePath := filepath.Join(dir, "scry_linux_amd64.tar.gz")
	os.WriteFile(archivePath, []byte("real content"), 0644)

	checksumsPath := filepath.Join(dir, "checksums.txt")
	line := "deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef  scry_linux_amd64.tar.gz\n"
	os.WriteFile(checksumsPath, []byte(line), 0644)

	err := VerifyChecksum(archivePath, "scry_linux_amd64.tar.gz", checksumsPath)
	if !errors.Is(err, ErrChecksumMismatch) {
		t.Fatalf("expected ErrChecksumMismatch, got %v", err)
	}
}

func TestVerifyChecksum_MissingEntry(t *testing.T) {
	dir := t.TempDir()

	archivePath := filepath.Join(dir, "scry_linux_amd64.tar.gz")
	os.WriteFile(archivePath, []byte("content"), 0644)

	checksumsPath := filepath.Join(dir, "checksums.txt")
	line := "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890  other_file.tar.gz\n"
	os.WriteFile(checksumsPath, []byte(line), 0644)

	err := VerifyChecksum(archivePath, "scry_linux_amd64.tar.gz", checksumsPath)
	if !errors.Is(err, ErrChecksumMismatch) {
		t.Fatalf("expected ErrChecksumMismatch, got %v", err)
	}
}

func TestVerifyChecksum_MalformedFile(t *testing.T) {
	dir := t.TempDir()

	archivePath := filepath.Join(dir, "scry_linux_amd64.tar.gz")
	os.WriteFile(archivePath, []byte("content"), 0644)

	checksumsPath := filepath.Join(dir, "checksums.txt")
	os.WriteFile(checksumsPath, []byte("this is not a valid checksums file\n"), 0644)

	err := VerifyChecksum(archivePath, "scry_linux_amd64.tar.gz", checksumsPath)
	if !errors.Is(err, ErrChecksumMismatch) {
		t.Fatalf("expected ErrChecksumMismatch, got %v", err)
	}
}
