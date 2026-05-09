//go:build !windows

package update

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ReplaceBinary extracts the scry binary from a tar.gz archive and
// atomically replaces the executable at binaryPath.
func ReplaceBinary(archivePath, binaryPath string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrReplaceFailed, err)
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrReplaceFailed, err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return fmt.Errorf("%w: binary not found in archive", ErrReplaceFailed)
		}
		if err != nil {
			return fmt.Errorf("%w: %v", ErrReplaceFailed, err)
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		// GoReleaser puts the binary at root level named "scry"
		if filepath.Base(hdr.Name) != "scry" {
			continue
		}

		return extractAndReplace(tr, binaryPath)
	}
}

func extractAndReplace(r io.Reader, binaryPath string) error {
	dir := filepath.Dir(binaryPath)
	tmp, err := os.CreateTemp(dir, ".scry-update-*")
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
	if err = os.Chmod(tmpPath, 0755); err != nil {
		return fmt.Errorf("%w: %v", ErrReplaceFailed, err)
	}
	if err = os.Rename(tmpPath, binaryPath); err != nil {
		return fmt.Errorf("%w: %v", ErrReplaceFailed, err)
	}
	return nil
}
