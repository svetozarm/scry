# PRD: Self-Update

## 1. Purpose

Scry is distributed as a single binary via GitHub Releases (built by GoReleaser). Currently, users have no way to know a new version is available or to update without manually downloading a release. This feature adds a `scry update` command that checks the GitHub Releases API for a newer version, downloads the appropriate platform binary, verifies its integrity via SHA-256 checksum, and replaces the running binary in-place.

**Problem:** Users run stale versions indefinitely because there is no update notification or mechanism built into the CLI.

**Vision:** A single command (`scry update`) brings the user to the latest release with zero external tooling required.

**Success Metrics:**
- 90% of active users are on the latest release within 7 days of a new tag.
- Zero reported cases of corrupted binaries after update (checksum verification prevents this).
- Update completes in under 30 seconds on a 10 Mbps connection for typical binary sizes (~12 MB).

## 2. Personas

| Persona | Description |
|---|---|
| **Developer (primary)** | Uses Scry daily to generate commit messages. Wants the latest features and fixes without leaving the terminal or managing package managers. |
| **CI Operator** | Runs Scry in non-interactive pipelines. Needs deterministic version pinning but may use `scry update` in setup scripts to pull the latest. |

## 3. Functional Requirements

### 3.1 Ubiquitous Requirements

- REQ-U-001: The system shall embed the current version string (set via `-ldflags` at build time) in the binary.
- REQ-U-002: The system shall derive the expected release asset name using the pattern `scry_<version>_<GOOS>_<GOARCH>.tar.gz` (or `.zip` for Windows), matching the GoReleaser `name_template`.
- REQ-U-003: The system shall use the GitHub repository `svetozarm/scry` as the release source.

### 3.2 Event-Driven Requirements

- REQ-E-001: When the user runs `scry update`, the system shall query the GitHub Releases API for the latest release tag.
- REQ-E-002: When the latest release tag is newer than the embedded version, the system shall download the platform-appropriate archive asset from the release.
- REQ-E-003: When the latest release tag is newer than the embedded version, the system shall download the `checksums.txt` asset from the release.
- REQ-E-004: When the archive is downloaded, the system shall compute the SHA-256 hash of the downloaded archive and compare it against the corresponding entry in `checksums.txt`.
- REQ-E-005: When the checksum matches, the system shall extract the binary from the archive and replace the current executable.
- REQ-E-006: When the update completes successfully, the system shall display the old version, the new version, and a confirmation message.
- REQ-E-007: When the latest release tag equals the embedded version, the system shall inform the user that Scry is already up to date and exit 0.

### 3.3 State-Driven Requirements

- REQ-S-001: While in non-interactive mode (`--non-interactive`), the system shall output plain text progress and result to stdout without styling or prompts.
- REQ-S-002: While the current binary's file path is not writable by the current user, the system shall report a clear error message indicating the permission issue and suggest running with elevated privileges.

### 3.4 Unwanted Behaviour Requirements

- REQ-X-001: If the GitHub Releases API is unreachable or returns a non-200 response, then the system shall display an error with the HTTP status and exit with code 5.
- REQ-X-002: If the checksum of the downloaded archive does not match the expected value in `checksums.txt`, then the system shall abort the update, delete the downloaded file, and display an integrity error.
- REQ-X-003: If the platform-appropriate asset is not found in the release, then the system shall display an error listing the current OS/architecture and exit with code 5.
- REQ-X-004: If the binary file is locked for writing (common on Windows where the running executable cannot be overwritten), then the system shall rename the current binary to a temporary name in the same directory, write the new binary to the original path, and delete the renamed old binary on success.
- REQ-X-005: If the rename-and-replace strategy fails (e.g., directory not writable), then the system shall display an error explaining the failure and suggest manual replacement.
- REQ-X-006: If the embedded version string is not set (development build), then the system shall display a warning that update is unavailable for development builds and exit with code 0.
- REQ-X-007: If the download is interrupted (network failure mid-transfer), then the system shall clean up any partial temporary files and display a retry suggestion.

### 3.5 Optional Feature Requirements

None for this release.

## 4. Non-Functional Requirements

- REQ-NF-001: The system shall complete the version check (API call only, no download) in under 3 seconds on a typical connection.
- REQ-NF-002: The system shall download assets to a temporary file in the same directory as the binary (to ensure same-filesystem atomic rename).
- REQ-NF-003: The system shall not send any data to GitHub beyond the standard Releases API GET request (no telemetry, no user identification).
- REQ-NF-004: The system shall set the executable permission bits (`0755`) on the new binary after extraction (on Unix systems).
- REQ-NF-005: The system shall support all platforms produced by the GoReleaser config: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64, windows/arm64.
- REQ-NF-006: The system shall use the unauthenticated GitHub Releases API (no token required for public repos) to avoid configuration burden.

## 5. Constraints and Assumptions

**Constraints:**
- The GitHub repository is public, so the Releases API is accessible without authentication.
- GoReleaser produces a `checksums.txt` file containing SHA-256 hashes for all archives (this is the current configuration).
- The binary version must be injected at build time via `-ldflags "-X main.version=<tag>"` (or equivalent). GoReleaser handles this automatically; the local Makefile must be updated.
- No external dependencies beyond Go's standard library and `net/http` for the update module (no GitHub SDK).

**Assumptions:**
- The user has network access to `api.github.com` and `github.com` (for asset downloads).
- The user has write permission to the directory containing the Scry binary (or can escalate).
- Git tags follow semantic versioning prefixed with `v` (e.g., `v1.2.3`).
- The running binary can determine its own path via `os.Executable()`.

**Timeline:**
- Single delivery milestone. No phased rollout.

## 6. Out of Scope

- Automatic background update checks (no "a new version is available" on every run).
- Rollback to a previous version.
- Updating to a specific version (always updates to latest).
- Signature verification beyond SHA-256 checksum (no GPG/cosign).
- GitHub API authentication or private repository support.
- Self-update for binaries installed via package managers (Homebrew, apt, etc.) — those users should use their package manager.

## 7. Open Questions

| # | Question | Impact |
|---|---|---|
| 1 | Should `scry update` require explicit user confirmation before applying the update in interactive mode (e.g., "Update from v1.0.0 to v1.1.0? [Y/n]")? | Affects REQ-E-005 — may need an additional event-driven requirement for the confirmation prompt. |
| 2 | Should the version string include the git commit SHA for development builds (e.g., `v1.0.0-dev+abc1234`) to distinguish from release builds? | Affects REQ-X-006 detection logic. |
| 3 | On Windows, should the old binary be scheduled for deletion on next reboot if immediate deletion fails? | Affects REQ-X-004 implementation complexity. |
