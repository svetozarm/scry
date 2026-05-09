package update

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWritable_WritableDir(t *testing.T) {
	dir := t.TempDir()
	if !writable(dir) {
		t.Fatal("expected writable dir to return true")
	}
}

func TestWritable_NonExistentDir(t *testing.T) {
	if writable("/nonexistent_path_xyz_12345") {
		t.Fatal("expected non-existent dir to return false")
	}
}

func TestWritable_ReadOnlyDir(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping: root can write to read-only dirs")
	}
	dir := t.TempDir()
	readOnly := filepath.Join(dir, "readonly")
	if err := os.Mkdir(readOnly, 0o555); err != nil {
		t.Fatal(err)
	}
	if writable(readOnly) {
		t.Fatal("expected read-only dir to return false")
	}
}
