package update

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsTarGz(t *testing.T) {
	assert.True(t, isTarGz("scry_1.0.0_linux_amd64.tar.gz"))
	assert.True(t, isTarGz("/tmp/download/archive.tar.gz"))
	assert.False(t, isTarGz("archive.zip"))
	assert.False(t, isTarGz("archive.tar"))
	assert.False(t, isTarGz(""))
}

func TestIsZip(t *testing.T) {
	assert.True(t, isZip("scry_1.0.0_windows_amd64.zip"))
	assert.True(t, isZip("/tmp/download/archive.zip"))
	assert.False(t, isZip("archive.tar.gz"))
	assert.False(t, isZip("archive.tar"))
	assert.False(t, isZip(""))
}

func TestDetectArchiveFormat(t *testing.T) {
	format, err := detectArchiveFormat("archive.tar.gz")
	assert.NoError(t, err)
	assert.Equal(t, archiveTarGz, format)

	format, err = detectArchiveFormat("archive.zip")
	assert.NoError(t, err)
	assert.Equal(t, archiveZip, format)

	_, err = detectArchiveFormat("archive.tar")
	assert.ErrorIs(t, err, ErrReplaceFailed)

	_, err = detectArchiveFormat("noextension")
	assert.ErrorIs(t, err, ErrReplaceFailed)
}
