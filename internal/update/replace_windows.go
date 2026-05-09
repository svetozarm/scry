//go:build windows

package update

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ReplaceBinary extracts the scry.exe binary from a zip archive and
// replaces the executable at binaryPath using rename-and-replace.
func ReplaceBinary(archivePath, binaryPath string) error {
	zr, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrReplaceFailed, err)
	}
	defer zr.Close()

	for _, f := range zr.File {
		if filepath.Base(f.Name) != "scry.exe" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("%w: %v", ErrReplaceFailed, err)
		}
		defer rc.Close()
		return extractAndReplaceWindows(rc, binaryPath)
	}
	return fmt.Errorf("%w: binary not found in archive", ErrReplaceFailed)
}

func extractAndReplaceWindows(r io.Reader, binaryPath string) error {
	dir := filepath.Dir(binaryPath)
	tmp, err := os.CreateTemp(dir, ".scry-update-*.exe")
	if err != nil {
		return fmt.Errorf("%w: %v", ErrReplaceFailed, err)
	}
	tmpPath := tmp.Name()
	defer func() {
		if err != nil {
			os.Remove(tmpPath)
		}
	}()

	if _, err = io.Copy(tmp, r); err != nil {
		tmp.Close()
		return fmt.Errorf("%w: %v", ErrReplaceFailed, err)
	}
	if err = tmp.Close(); err != nil {
		return fmt.Errorf("%w: %v", ErrReplaceFailed, err)
	}

	oldPath := binaryPath + ".old"
	os.Remove(oldPath) // best-effort remove stale .old

	if err = os.Rename(binaryPath, oldPath); err != nil {
		return fmt.Errorf("%w: try running with elevated privileges or replace manually: %v", ErrReplaceFailed, err)
	}
	if err = os.Rename(tmpPath, binaryPath); err != nil {
		// Attempt rollback
		os.Rename(oldPath, binaryPath)
		return fmt.Errorf("%w: try running with elevated privileges or replace manually: %v", ErrReplaceFailed, err)
	}

	os.Remove(oldPath) // best-effort cleanup
	return nil
}
