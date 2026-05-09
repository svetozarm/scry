package update

import "context"

// Result holds the outcome of an update check or execution.
type Result struct {
	OldVersion string
	NewVersion string
	UpToDate   bool
}

// Run executes the full update workflow.
func Run(ctx context.Context, currentVersion string, checker ReleaseChecker) (*Result, error) {
	if currentVersion == "" {
		return nil, ErrDevBuild
	}

	release, err := checker.LatestRelease(ctx)
	if err != nil {
		return nil, err
	}

	if !isNewer(release.Tag, currentVersion) {
		return &Result{UpToDate: true}, nil
	}

	return &Result{OldVersion: currentVersion, NewVersion: release.Tag}, nil
}
