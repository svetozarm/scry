package main

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
)

func TestStaticBinary(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("static binary verification only runs on Linux")
	}

	binary := t.TempDir() + "/scry"

	// Build with static flags
	build := exec.Command("go", "build", "-ldflags=-s -w", "-o", binary, ".")
	build.Env = append(os.Environ(), "CGO_ENABLED=0")
	out, err := build.CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %s\n%s", err, out)
	}

	// Verify binary exists and is non-empty
	info, err := os.Stat(binary)
	if err != nil {
		t.Fatalf("binary not found: %s", err)
	}
	if info.Size() == 0 {
		t.Fatal("binary is empty")
	}

	// Verify statically linked via ldd
	ldd := exec.Command("ldd", binary)
	lddOut, _ := ldd.CombinedOutput()
	lddStr := string(lddOut)
	if !strings.Contains(lddStr, "not a dynamic executable") && !strings.Contains(lddStr, "statically linked") {
		t.Errorf("ldd indicates dynamic linking: %s", lddStr)
	}
}
