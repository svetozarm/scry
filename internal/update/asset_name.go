package update

import (
	"runtime"
	"strings"
)

// deriveAssetName builds the expected release asset filename for the current platform.
func deriveAssetName(tag string) string {
	version := strings.TrimPrefix(tag, "v")
	ext := ".tar.gz"
	if runtime.GOOS == "windows" {
		ext = ".zip"
	}
	return "scry_" + version + "_" + runtime.GOOS + "_" + runtime.GOARCH + ext
}
