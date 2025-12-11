# Feature Specification: Boarding CLI Tool

**Feature Branch**: `002-boarding-cli`
**Created**: 2025-12-10
**Status**: Draft
**Input**: User description: "Add a small CLI tool (`boarding`) for developers and CI to interact with the BoardingPass service. The CLI should offer a number of commands: pass, info, connections, load, command, complete."

## Clarifications

### Session 2025-12-10

- Q: How should the CLI handle TLS certificate validation for self-signed certificates? → A: Prompt user on first connection to accept/reject certificate fingerprint, then remember choice. Also support `--ca-cert` flag to specify a CA bundle for certificate verification if not in system's CA bundle.
- Q: How should users specify BoardingPass service connection details (host, port)? → A: Support multiple methods with precedence: command-line flags > environment variables > config file. Config file is `config.yaml` located in OS-specific user config directory (e.g., `~/.config/boardingpass/` on Linux).
- Q: Where should session tokens be stored and what security measures should apply? → A: Store in OS-specific temporary directory with restrictive file permissions (0600), allowing tokens to persist across CLI invocations but not system reboots.
- Q: What structure should the config.yaml file use? → A: Flat structure with keys at root level (e.g., `host: boardingpass.local`, `port: 8443`).
- Q: What should happen when no connection details are provided via any method? → A: Error immediately with a clear message indicating connection details are required.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Authentication (Priority: P1)

A developer or CI system needs to authenticate with a BoardingPass service before performing any operations. They can provide credentials interactively (for manual use) or via flags (for automated scripts and CI).

**Why this priority**: Authentication is the foundation for all other operations. Without the ability to login, no other commands can function. This represents the minimum viable CLI tool.

**Independent Test**: Can be fully tested by running `boarding pass` with valid credentials and verifying that a session token is obtained and stored for subsequent commands.

**Acceptance Scenarios**:

1. **Given** the BoardingPass service is running, **When** a user runs `boarding pass` without flags, **Then** the CLI prompts for username and password, authenticates via SRP-6a, and stores the session token
2. **Given** the BoardingPass service is running, **When** a user runs `boarding pass --username admin --password secret`, **Then** the CLI authenticates without prompting and stores the session token
3. **Given** invalid credentials are provided, **When** authentication is attempted, **Then** the CLI displays a clear error message and exits with a non-zero code

---

### User Story 2 - Query System Information (Priority: P2)

A developer or CI system needs to retrieve hardware and system information from a device to verify inventory or check device state before provisioning.

**Why this priority**: This is the most basic read-only operation that provides value immediately after authentication. It allows users to inspect device state without making any changes.

**Independent Test**: Can be fully tested by authenticating with `boarding pass`, then running `boarding info` and verifying that system information (CPU, board, TPM, OS, FIPS status) is displayed in the requested format.

**Acceptance Scenarios**:

1. **Given** an authenticated session exists, **When** a user runs `boarding info` or `boarding info -o yaml`, **Then** the CLI queries the `/info` endpoint and displays system information in YAML format
2. **Given** an authenticated session exists, **When** a user runs `boarding info -o json`, **Then** the CLI displays system information in JSON format
3. **Given** no authenticated session exists, **When** a user runs `boarding info`, **Then** the CLI displays an authentication error and exits with a non-zero code

---

### User Story 3 - Query Network Interfaces (Priority: P3)

A developer or CI system needs to inspect network interface configuration to verify connectivity or determine which interfaces are available for provisioning.

**Why this priority**: Like system information, this is a read-only query operation that provides inspection capabilities. It's slightly lower priority than system info as network details are typically less critical for initial device verification.

**Independent Test**: Can be fully tested by authenticating with `boarding pass`, then running `boarding connections` and verifying that network interface information is displayed in the requested format.

**Acceptance Scenarios**:

1. **Given** an authenticated session exists, **When** a user runs `boarding connections` or `boarding connections -o yaml`, **Then** the CLI queries the `/network` endpoint and displays network interface information in YAML format
2. **Given** an authenticated session exists, **When** a user runs `boarding connections -o json`, **Then** the CLI displays network interface information in JSON format
3. **Given** no authenticated session exists, **When** a user runs `boarding connections`, **Then** the CLI displays an authentication error and exits with a non-zero code

---

### User Story 4 - Upload Configuration (Priority: P4)

A developer or CI system needs to provision device configuration by uploading a directory of configuration files to the device. This enables automated device setup and configuration management.

**Why this priority**: This is the first write operation and represents a critical provisioning capability. However, it requires prior authentication and is more complex than read operations, hence the lower priority.

**Independent Test**: Can be fully tested by authenticating with `boarding pass`, preparing a directory with test configuration files, running `boarding load /path/to/config`, and verifying that the files are uploaded and the service confirms successful provisioning.

**Acceptance Scenarios**:

1. **Given** an authenticated session exists and a configuration directory exists, **When** a user runs `boarding load /path/to/config`, **Then** the CLI uploads all files in the directory to the `/configure` endpoint and displays success confirmation
2. **Given** the configuration directory does not exist, **When** a user runs `boarding load /nonexistent/path`, **Then** the CLI displays a clear error message about the missing directory and exits with a non-zero code
3. **Given** the upload is in progress, **When** files are being sent, **Then** the CLI displays progress feedback to indicate the operation status
4. **Given** an authenticated session exists but the service rejects the configuration, **When** upload completes, **Then** the CLI displays the service's error message and exits with a non-zero code

---

### User Story 5 - Execute Commands (Priority: P5)

A developer or CI system needs to execute administrative commands on the device (from an allow-list) to perform system operations during provisioning or troubleshooting.

**Why this priority**: Command execution is a privileged operation with security implications. While important for device management, it's lower priority than configuration upload because many provisioning scenarios can be completed with configuration files alone.

**Independent Test**: Can be fully tested by authenticating with `boarding pass`, running `boarding command "systemctl status networking"` and verifying that the command executes on the device and output is displayed.

**Acceptance Scenarios**:

1. **Given** an authenticated session exists, **When** a user runs `boarding command "systemctl restart networking"`, **Then** the CLI sends the command to the `/command` endpoint and displays the command output
2. **Given** an authenticated session exists but the command is not allow-listed, **When** a user runs `boarding command "rm -rf /"`, **Then** the CLI displays an error indicating the command is not permitted
3. **Given** no authenticated session exists, **When** a user runs `boarding command "ls"`, **Then** the CLI displays an authentication error and exits with a non-zero code
4. **Given** command execution fails on the device, **When** the command is sent, **Then** the CLI displays the error output and exits with a non-zero code

---

### User Story 6 - Logout and Complete Session (Priority: P6)

A developer or CI system needs to explicitly terminate their session after completing provisioning operations, triggering the BoardingPass service to create the sentinel file and prepare for shutdown.

**Why this priority**: This is the final cleanup step in the provisioning workflow. While important for proper lifecycle management, all other operations must work before logout is meaningful.

**Independent Test**: Can be fully tested by authenticating with `boarding pass`, performing any operation, then running `boarding complete` and verifying that the session is terminated and the service acknowledges completion.

**Acceptance Scenarios**:

1. **Given** an authenticated session exists, **When** a user runs `boarding complete`, **Then** the CLI calls the `/complete` endpoint, the session is terminated, and success is confirmed
2. **Given** no authenticated session exists, **When** a user runs `boarding complete`, **Then** the CLI displays a message indicating no active session and exits successfully
3. **Given** the complete operation succeeds, **When** the CLI exits, **Then** any stored session token is removed

---

### Edge Cases

- What happens when the network connection to the BoardingPass service is lost mid-operation?
- What happens when a user rejects a certificate fingerprint on first connection?
- What happens when a session token expires (30-min TTL) during a long-running operation?
- How does the CLI behave when multiple instances try to use the same session?
- What happens when the BoardingPass service is unavailable or unreachable?
- What happens when the config file is malformed or contains invalid YAML?
- How does the CLI handle very large configuration directories?
- What happens when output formatting is requested but the API returns malformed data?
- How does the CLI handle special characters or binary data in command output?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: CLI MUST support a `pass` command that authenticates users via the BoardingPass SRP-6a authentication protocol
- **FR-002**: CLI MUST support interactive credential input (prompting for username and password) when `pass` is called without flags
- **FR-003**: CLI MUST support non-interactive credential input via `--username` and `--password` flags for the `pass` command
- **FR-004**: CLI MUST store session tokens in the OS-specific temporary directory after successful authentication for use by subsequent commands
- **FR-005**: CLI MUST set file permissions to 0600 (owner read/write only) on session token files to prevent unauthorized access
- **FR-006**: CLI MUST store session tokens in a per-user temporary directory to prevent conflicts between multiple users or CLI instances
- **FR-007**: CLI MUST support an `info` command that queries the `/info` API endpoint and returns system inventory information
- **FR-008**: CLI MUST support a `connections` command that queries the `/network` API endpoint and returns network interface information
- **FR-009**: CLI MUST support YAML output format as the default for `info` and `connections` commands
- **FR-010**: CLI MUST support JSON output format via `-o json` flag for `info` and `connections` commands
- **FR-011**: CLI MUST support a `load` command that accepts a directory path and uploads all files to the `/configure` API endpoint
- **FR-012**: CLI MUST validate that the specified configuration directory exists before attempting upload
- **FR-013**: CLI MUST support a `command` command that accepts a command string and executes it via the `/command` API endpoint
- **FR-014**: CLI MUST support a `complete` command that terminates the session via the `/complete` API endpoint
- **FR-015**: CLI MUST verify that a valid session exists before executing commands that require authentication
- **FR-016**: CLI MUST display clear error messages when authentication fails or session is invalid
- **FR-017**: CLI MUST exit with appropriate status codes (0 for success, non-zero for errors)
- **FR-018**: CLI MUST support specification of BoardingPass service connection details via command-line flags (`--host`, `--port`)
- **FR-019**: CLI MUST support specification of BoardingPass service connection details via environment variables (`BOARDING_HOST`, `BOARDING_PORT`)
- **FR-020**: CLI MUST support specification of BoardingPass service connection details via a `config.yaml` file in the OS-specific user configuration directory
- **FR-021**: CLI MUST use a flat YAML structure in config.yaml with keys at the root level (e.g., `host: value`, `port: value`)
- **FR-022**: CLI MUST use the OS-specific default user configuration directory for storing configuration (e.g., `~/.config/boardingpass/` on Linux, `~/Library/Application Support/boardingpass/` on macOS, `%APPDATA%\boardingpass\` on Windows)
- **FR-023**: CLI MUST apply configuration precedence in order: command-line flags > environment variables > config file
- **FR-024**: CLI MUST display a clear error message and exit with non-zero status when no connection details are provided via any method
- **FR-025**: CLI MUST support TLS/HTTPS connections to the BoardingPass service
- **FR-026**: CLI MUST prompt the user on first connection to a BoardingPass service to accept or reject the TLS certificate fingerprint
- **FR-027**: CLI MUST remember accepted certificate fingerprints for subsequent connections to the same service
- **FR-028**: CLI MUST support a `--ca-cert` flag to specify a custom CA bundle file for certificate verification
- **FR-029**: CLI MUST validate certificates against the system trust store or custom CA bundle when provided
- **FR-030**: CLI MUST fail the connection and display a clear error if certificate validation fails and the user has not accepted the fingerprint
- **FR-031**: CLI MUST handle expired session tokens gracefully by prompting for re-authentication
- **FR-032**: CLI MUST display progress feedback for long-running operations (file uploads, command execution)
- **FR-033**: CLI MUST clean up stored session tokens when `complete` command is executed successfully

### Key Entities

- **Session Token**: A cryptographic token issued by the BoardingPass service after successful SRP-6a authentication, valid for 30 minutes, used to authorize subsequent API requests; stored in OS-specific temporary directory with restrictive permissions (0600) to persist across CLI invocations but not system reboots
- **Configuration Bundle**: A collection of files from a directory that are uploaded atomically to the BoardingPass service for device provisioning
- **Command**: A string representing a system command to be executed on the device, subject to allow-list validation by the BoardingPass service
- **Configuration File**: A YAML file (`config.yaml`) stored in the OS-specific user configuration directory with a flat structure containing connection details (e.g., `host: boardingpass.local`, `port: 8443`) and other CLI preferences; read during CLI startup with lower precedence than flags and environment variables

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Developers can authenticate with a BoardingPass service in under 5 seconds (interactive or non-interactive mode)
- **SC-002**: Developers can query system information and receive formatted output in under 2 seconds
- **SC-003**: CI systems can complete a full provisioning workflow (authenticate, query info, upload config, execute commands, complete) without manual intervention
- **SC-004**: Configuration uploads complete within 30 seconds for directories containing up to 100 files totaling 10MB
- **SC-005**: All CLI operations provide clear error messages that enable users to diagnose and fix issues without consulting documentation
- **SC-006**: 95% of users successfully complete their first provisioning task without errors (excluding infrastructure/network issues)
- **SC-007**: Session management works transparently - users do not need to manually handle session tokens or renewal

## Assumptions *(optional)*

- The BoardingPass service is running and accessible over the network
- Users have valid credentials (username and password) for the BoardingPass service
- The BoardingPass service may use self-signed TLS certificates or CA-signed TLS certificates; CLI users may have the CA certs in their system CA bundle or pass via the `--ca-cert` flag
- Certificate fingerprint acceptance choices are stored persistently (per service endpoint) for convenience across CLI invocations
- Configuration directories contain only text-based configuration files (not large binaries)
- The CLI will be used primarily in trusted environments (developer workstations, CI systems) rather than public networks
- Session tokens are stored in OS-specific temporary directories (e.g., `/tmp/` on Linux) with restrictive file permissions and do not persist across system reboots
- Each user has their own temporary directory for session tokens to prevent conflicts between concurrent CLI instances
- The BoardingPass service's `/configure` endpoint accepts a specific file bundle format (to be determined during implementation planning)
- Users running the CLI have appropriate network access and firewall rules configured to reach the BoardingPass service
