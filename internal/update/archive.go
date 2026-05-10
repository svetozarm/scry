package update

import (
	"fmt"
	"strings"
)

type archiveFormat int

const (
	archiveTarGz archiveFormat = iota
	archiveZip
)

func isTarGz(path string) bool { return strings.HasSuffix(path, ".tar.gz") }
func isZip(path string) bool   { return strings.HasSuffix(path, ".zip") }

func detectArchiveFormat(path string) (archiveFormat, error) {
	switch {
	case isTarGz(path):
		return archiveTarGz, nil
	case isZip(path):
		return archiveZip, nil
	default:
		return 0, fmt.Errorf("%w: unsupported archive format: %s", ErrReplaceFailed, path)
	}
}
