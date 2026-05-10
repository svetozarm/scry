package update

import (
	"errors"
	"fmt"
	"testing"
)

func TestSentinelErrors_AreMatchableViaErrorsIs(t *testing.T) {
	sentinels := []error{
		ErrUpdateAPI,
		ErrChecksumMismatch,
		ErrAssetNotFound,
		ErrPermission,
		ErrReplaceFailed,
		ErrDevBuild,
	}

	for _, sentinel := range sentinels {
		wrapped := fmt.Errorf("context: %w", sentinel)
		if !errors.Is(wrapped, sentinel) {
			t.Errorf("errors.Is failed for wrapped %q", sentinel)
		}
	}
}
