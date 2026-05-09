# TODO: Self-Update Feature

## Phase 1: Foundation (version embedding + errors)

- [x] 1.1 Add `var version string` to `main.go`, pass to `cmd` package
- [x] 1.2 Update `Makefile` to inject version via `-ldflags "-X main.version=..."`
- [x] 1.3 Update `.goreleaser.yaml` to inject version via ldflags
- [x] 1.4 Create `internal/update/errors.go` with sentinel errors
- [x] 1.5 Write tests for error type matching

## Phase 2: GitHub Client

- [x] 2.1 Create `internal/update/github.go` with `ReleaseChecker` interface
- [x] 2.2 Implement `GitHubClient` struct with `LatestRelease()`
- [x] 2.3 Implement `DownloadAsset()` with temp file + cleanup
- [x] 2.4 Write tests using `httptest.Server` (success, non-200, timeout, malformed JSON)

## Phase 3: Checksum Verification

- [x] 3.1 Create `internal/update/checksum.go` with `VerifyChecksum()`
- [x] 3.2 Implement checksums.txt parsing (GoReleaser format)
- [x] 3.3 Implement SHA-256 computation and comparison
- [x] 3.4 Write tests (match, mismatch, missing entry, malformed file)

## Phase 4: Binary Replacement

- [x] 4.1 Create `internal/update/replace.go` with Unix strategy (tar.gz extraction + atomic rename)
- [x] 4.2 Create `internal/update/replace_windows.go` with Windows rename-and-replace strategy
- [x] 4.3 Implement archive format detection (tar.gz vs zip by extension)
- [x] 4.4 Write tests (extraction, permissions, read-only directory failure)

## Phase 5: Update Orchestrator

- [x] 5.1 Create `internal/update/update.go` with `Run()` function
- [x] 5.2 Implement `isNewer()` semver comparison
- [x] 5.3 Implement `deriveAssetName()` using runtime.GOOS/GOARCH
- [x] 5.4 Implement write permission check
- [x] 5.5 Wire together: version check → download → verify → replace → cleanup
- [x] 5.6 Write orchestrator tests with mocked `ReleaseChecker`

## Phase 6: CLI Integration

- [ ] 6.1 Create `cmd/update.go` with `updateCmd` Cobra subcommand
- [ ] 6.2 Register `updateCmd` on `rootCmd`
- [ ] 6.3 Add update error mappings to `cmd/errors.go` (exit code 5)
- [ ] 6.4 Handle non-interactive mode (plain text output)
- [ ] 6.5 Handle `ErrDevBuild` as warning (exit 0)
- [ ] 6.6 Write CLI integration tests

## Phase 7: Verification & Cleanup

- [ ] 7.1 Run full test suite, verify all pass
- [ ] 7.2 Manual test: build with version, tag, run `scry update`
- [ ] 7.3 Verify cross-compilation still works for all 6 platforms
- [ ] 7.4 Update README with `scry update` documentation
