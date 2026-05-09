# BDD: Self-Update

```gherkin
Feature: Self-Update
  As a developer
  I want to update Scry to the latest version with a single command
  So that I always have the latest features and fixes without manual downloads

  Background:
    Given the Scry binary has an embedded version string
    And the binary is located at a known executable path

  # --- Ubiquitous Requirements ---

  @REQ-U-001
  Scenario: Version string is embedded in the binary
    When the user runs any Scry command
    Then the system has access to the embedded version string

  @REQ-U-002
  Scenario Outline: Asset name derived from platform
    Given the current OS is <os>
    And the current architecture is <arch>
    When the system determines the asset name for version "1.2.3"
    Then the expected asset name is "scry_1.2.3_<os>_<arch>.<ext>"

    Examples:
      | os      | arch  | ext    |
      | linux   | amd64 | tar.gz |
      | linux   | arm64 | tar.gz |
      | darwin  | amd64 | tar.gz |
      | darwin  | arm64 | tar.gz |
      | windows | amd64 | zip    |
      | windows | arm64 | zip    |

  @REQ-U-003
  Scenario: Release source is the correct GitHub repository
    When the system queries for the latest release
    Then the request is made to the GitHub Releases API for "svetozarm/scry"

  # --- Event-Driven Requirements ---

  @REQ-E-001 @smoke
  Scenario: Check for latest release
    Given the embedded version is "v1.0.0"
    And the latest GitHub release tag is "v1.1.0"
    When the user runs "scry update"
    Then the system queries the GitHub Releases API for the latest release tag

  @REQ-E-002
  Scenario: Download platform-appropriate archive when newer version exists
    Given the embedded version is "v1.0.0"
    And the latest GitHub release tag is "v1.1.0"
    And the current platform is "linux/amd64"
    When the user runs "scry update"
    Then the system downloads "scry_1.1.0_linux_amd64.tar.gz" from the release assets

  @REQ-E-003
  Scenario: Download checksums file when newer version exists
    Given the embedded version is "v1.0.0"
    And the latest GitHub release tag is "v1.1.0"
    When the user runs "scry update"
    Then the system downloads "checksums.txt" from the release assets

  @REQ-E-004
  Scenario: Verify archive integrity via SHA-256 checksum
    Given the system has downloaded the archive "scry_1.1.0_linux_amd64.tar.gz"
    And the system has downloaded "checksums.txt"
    When the system computes the SHA-256 hash of the archive
    Then the computed hash matches the entry for "scry_1.1.0_linux_amd64.tar.gz" in checksums.txt

  @REQ-E-005
  Scenario: Replace current binary after successful checksum verification
    Given the checksum verification has passed
    And the archive contains the new Scry binary
    When the system extracts the binary from the archive
    Then the current executable is replaced with the new binary

  @REQ-E-006
  Scenario: Display update confirmation
    Given the embedded version is "v1.0.0"
    And the update to "v1.1.0" completes successfully
    When the update finishes
    Then the system displays "v1.0.0" as the old version
    And the system displays "v1.1.0" as the new version
    And the system displays a success confirmation message

  @REQ-E-007
  Scenario: Already up to date
    Given the embedded version is "v1.1.0"
    And the latest GitHub release tag is "v1.1.0"
    When the user runs "scry update"
    Then the system displays a message that Scry is already up to date
    And the exit code is 0

  # --- State-Driven Requirements ---

  @REQ-S-001
  Scenario: Non-interactive mode outputs plain text
    Given the embedded version is "v1.0.0"
    And the latest GitHub release tag is "v1.1.0"
    When the user runs "scry update --non-interactive"
    Then the output is plain text without styling or prompts
    And progress information is written to stdout

  @REQ-S-002
  Scenario: Binary path is not writable
    Given the embedded version is "v1.0.0"
    And the latest GitHub release tag is "v1.1.0"
    And the current binary path is not writable by the current user
    When the user runs "scry update"
    Then the system displays an error indicating a permission issue
    And the system suggests running with elevated privileges

  # --- Unwanted Behaviour Requirements ---

  @REQ-X-001
  Scenario: GitHub API unreachable
    Given the GitHub Releases API is unreachable
    When the user runs "scry update"
    Then the system displays an error with the connection failure details
    And the exit code is 5

  @REQ-X-001
  Scenario: GitHub API returns non-200 response
    Given the GitHub Releases API returns HTTP status 503
    When the user runs "scry update"
    Then the system displays an error containing "503"
    And the exit code is 5

  @REQ-X-002
  Scenario: Checksum mismatch aborts update
    Given the system has downloaded the archive
    And the computed SHA-256 hash does not match the expected value in checksums.txt
    When the checksum verification runs
    Then the update is aborted
    And the downloaded archive is deleted
    And the system displays an integrity error

  @REQ-X-003
  Scenario: Platform asset not found in release
    Given the embedded version is "v1.0.0"
    And the latest GitHub release tag is "v1.1.0"
    And the release does not contain an asset for the current OS/architecture
    When the user runs "scry update"
    Then the system displays an error listing the current OS and architecture
    And the exit code is 5

  @REQ-X-004
  Scenario: Binary locked for writing on Windows
    Given the current platform is Windows
    And the running binary cannot be overwritten directly
    When the system attempts to replace the binary
    Then the system renames the current binary to a temporary name in the same directory
    And the system writes the new binary to the original path
    And the system deletes the renamed old binary

  @REQ-X-005
  Scenario: Rename-and-replace strategy fails
    Given the current platform is Windows
    And the running binary cannot be overwritten directly
    And the directory is not writable for rename operations
    When the system attempts the rename-and-replace strategy
    Then the system displays an error explaining the failure
    And the system suggests manual replacement

  @REQ-X-006
  Scenario: Development build without version string
    Given the embedded version string is not set
    When the user runs "scry update"
    Then the system displays a warning that update is unavailable for development builds
    And the exit code is 0

  @REQ-X-007
  Scenario: Download interrupted by network failure
    Given the system is downloading the archive
    When the network connection is lost mid-transfer
    Then any partial temporary files are cleaned up
    And the system displays a retry suggestion

  # --- Non-Functional Requirements ---

  @REQ-NF-002
  Scenario: Temporary files stored in same directory as binary
    Given the binary is located at "/usr/local/bin/scry"
    When the system downloads the update archive
    Then the temporary file is created in "/usr/local/bin/"

  @REQ-NF-004
  Scenario: Executable permissions set on Unix
    Given the current platform is Unix
    When the new binary is written to disk
    Then the file permissions are set to 0755

  @REQ-NF-006
  Scenario: No authentication required for API access
    When the system queries the GitHub Releases API
    Then the request does not include an Authorization header
```
