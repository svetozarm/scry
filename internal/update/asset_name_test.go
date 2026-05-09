package update

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeriveAssetName(t *testing.T) {
	name := deriveAssetName("v1.2.3")

	ext := ".tar.gz"
	if runtime.GOOS == "windows" {
		ext = ".zip"
	}
	expected := "scry_1.2.3_" + runtime.GOOS + "_" + runtime.GOARCH + ext
	assert.Equal(t, expected, name)
}

func TestDeriveAssetName_StripsVPrefix(t *testing.T) {
	name := deriveAssetName("v0.9.1")
	assert.Contains(t, name, "0.9.1")
	assert.NotContains(t, name, "v0.9.1")
}
