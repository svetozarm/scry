package update

import (
	"context"
	"os"
	"path/filepath"
)

// executablePath is a variable for testability.
var executablePath = os.Executable

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

	// Find platform asset and checksums
	assetName := deriveAssetName(release.Tag)
	var assetURL, checksumsURL string
	for _, a := range release.Assets {
		switch a.Name {
		case assetName:
			assetURL = a.URL
		case "checksums.txt":
			checksumsURL = a.URL
		}
	}
	if assetURL == "" || checksumsURL == "" {
		return nil, ErrAssetNotFound
	}

	// Resolve binary path and check permissions
	binaryPath, err := executablePath()
	if err != nil {
		return nil, ErrPermission
	}
	dir := filepath.Dir(binaryPath)
	if !writable(dir) {
		return nil, ErrPermission
	}

	// Download archive and checksums
	archiveDest := filepath.Join(dir, assetName)
	checksumsDest := filepath.Join(dir, "checksums.txt")
	cleanup := func() {
		os.Remove(archiveDest)
		os.Remove(checksumsDest)
	}

	if err := checker.DownloadAsset(ctx, assetURL, archiveDest); err != nil {
		cleanup()
		return nil, err
	}
	if err := checker.DownloadAsset(ctx, checksumsURL, checksumsDest); err != nil {
		cleanup()
		return nil, err
	}

	// Verify and replace
	if err := VerifyChecksum(archiveDest, assetName, checksumsDest); err != nil {
		cleanup()
		return nil, err
	}
	if err := ReplaceBinary(archiveDest, binaryPath); err != nil {
		cleanup()
		return nil, err
	}

	cleanup()
	return &Result{OldVersion: currentVersion, NewVersion: release.Tag}, nil
}
