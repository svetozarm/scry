//go:build integration

package integration

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStartupLatency(t *testing.T) {
	const runs = 5
	durations := make([]time.Duration, runs)

	for i := range runs {
		start := time.Now()
		cmd := exec.Command(binaryPath, "--help")
		_ = cmd.Run()
		durations[i] = time.Since(start)
	}

	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	median := durations[runs/2]

	t.Logf("startup latencies: %v, median: %v", durations, median)
	assert.Less(t, median, 200*time.Millisecond, "median startup latency should be under 200ms")
}

func TestMemoryUsage(t *testing.T) {
	timeBin, err := exec.LookPath("time")
	require.NoError(t, err, "GNU time not found in PATH")

	dir := setupGitRepo(t)

	// Stage a ~100KB diff
	bigContent := strings.Repeat("// line of code\n", 6250) // ~100KB
	require.NoError(t, os.WriteFile(dir+"/big.txt", []byte(bigContent), 0644))
	c := exec.Command("git", "add", "big.txt")
	c.Dir = dir
	require.NoError(t, c.Run())

	cfgPath := writeConfig(t, dir, "provider: test\nmodel_id: m\n")

	cmd := exec.Command(timeBin, "-v", binaryPath, "--non-interactive", "--config", cfgPath)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "TEST_PROVIDER_RESPONSE=feat: big change")
	out, _ := cmd.CombinedOutput()

	// Parse "Maximum resident set size (kbytes): NNNN"
	var maxRSSKB int64
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, "Maximum resident set size") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				maxRSSKB, _ = strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
			}
		}
	}

	maxRSSMB := float64(maxRSSKB) / 1024.0
	t.Logf("max RSS: %d KB (%.1f MB)", maxRSSKB, maxRSSMB)

	require.Greater(t, maxRSSKB, int64(0), "failed to read max RSS from time -v output:\n%s", string(out))
	assert.Less(t, maxRSSMB, 50.0, fmt.Sprintf("memory usage %.1f MB exceeds 50 MB limit", maxRSSMB))
}
