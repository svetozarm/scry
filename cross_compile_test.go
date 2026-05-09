package main

import (
	"fmt"
	"os"
	"os/exec"
	"testing"
)

func TestCrossCompilation(t *testing.T) {
	targets := []struct {
		goos, goarch, suffix string
	}{
		{"linux", "amd64", ""},
		{"linux", "arm64", ""},
		{"darwin", "amd64", ""},
		{"darwin", "arm64", ""},
		{"windows", "amd64", ".exe"},
		{"windows", "arm64", ".exe"},
	}

	for _, tgt := range targets {
		t.Run(fmt.Sprintf("%s/%s", tgt.goos, tgt.goarch), func(t *testing.T) {
			binary := fmt.Sprintf("%s/scry-%s-%s%s", t.TempDir(), tgt.goos, tgt.goarch, tgt.suffix)

			build := exec.Command("go", "build", "-ldflags=-s -w", "-o", binary, ".")
			build.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS="+tgt.goos, "GOARCH="+tgt.goarch)
			out, err := build.CombinedOutput()
			if err != nil {
				t.Fatalf("cross-compile failed for %s/%s: %s\n%s", tgt.goos, tgt.goarch, err, out)
			}

			info, err := os.Stat(binary)
			if err != nil {
				t.Fatalf("binary not found: %s", err)
			}
			if info.Size() == 0 {
				t.Fatal("binary is empty")
			}
		})
	}
}
