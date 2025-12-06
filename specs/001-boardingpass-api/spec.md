# Feature Specification: BoardingPass API

**Feature Branch**: `001-boardingpass-api`
**Created**: 2025-12-05
**Status**: Draft
**Input**: User description: "Design and specify a lightweight, ephemeral component (`boardingpass`) for the headless Linux device that exposes a RESTful API over HTTPS (TLS 1.3 required). Authentication relies on SRP (Secure Remote Password) protocol to establish a session using a pre-shared secret, avoiding PKI complexity. The API allows querying device inventory and network state, uploading configuration bundles in a single transaction, and executing allow-listed commands. The component runs as a non-privileged user, manages its own lifecycle (timeout/sentinel file), logs to journald (redacting secrets), and must be packaged for RPM/DEB distributions."

## Clarifications

### Session 2025-12-06

- Q: Session Token Expiration Policy - The spec mentions that Session Tokens are "short-lived" but doesn't specify the exact expiration time. This affects both security posture and user experience during multi-step provisioning workflows. → A: Session tokens expire after 30 minutes
- Q: Configuration Bundle Path Restrictions - The spec requires configuration bundles to write files to `/etc`, but doesn't specify which subdirectories or files are permitted. Unrestricted write access to `/etc` could allow overwriting critical system files (e.g., `/etc/passwd`, `/etc/sudoers`). → A: Allow-list specific subdirectories in `/etc` (e.g., `/etc/systemd/`, `/etc/myapp/`)
- Q: Brute Force Protection Strategy - The spec's edge cases mention that repeated failed SRP authentication attempts should implement delay/backoff, but the exact mechanism isn't specified. This affects security posture and user experience when legitimate users make mistakes. → A: Progressive delay after failed attempts (1s, 2s, 5s, then reject for 60s)
- Q: TLS Certificate Source - The spec requires HTTPS with TLS 1.3, but doesn't specify how the TLS certificate is obtained or managed. This impacts deployment complexity, security, and whether external dependencies are required. → A: Self-signed certificate generated at first boot, if not provided in OS image
- Q: Provisioning Completion Trigger - The spec mentions creating the sentinel file "upon successful completion of the provisioning workflow," but doesn't specify what event triggers this. This affects the API design and operator workflow. → A: Explicit API endpoint (POST /complete) that operator calls to signal completion

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Secure Session Establishment (SRP) (Priority: P1)

A bootstrap operator needs to authenticate with the device using a shared secret (e.g., PIN/password on device label) over an untrusted network connection to establish a secure session without requiring client certificates or prior PKI setup.

****Why this priority**:** Security is the prerequisite for all other operations. SRP allows mutual authentication and perfect forward secrecy using a low-entropy password, which fits the "frictionless" and "minimal dependency" principles better than managing mTLS certificates for a one-time bootstrap.

****Independent Test**:** Can be fully tested by performing the SRP handshake against the API endpoints and verifying a valid Session Token is issued.

**Acceptance Scenarios**:

1. **Given** an unauthenticated client and a device with a stored SRP verifier, **When** the client initiates an SRP-6a handshake (POST /auth/srp/init) with the correct username, **Then** the server returns the salt and ephemeral value 'B'.
2. **Given** an initialized handshake, **When** the client sends a valid Session Proof (M1) to POST /auth/srp/verify, **Then** the server returns the Server Proof (M2) and a valid Session Token.
3. **Given** an incorrect password/proof, **When** the client attempts to verify, **Then** the server rejects the request with a 401 error and no Session Token is issued.
4. **Given** multiple failed authentication attempts (3 consecutive failures), **When** the client attempts a fourth authentication, **Then** the server rejects all authentication requests for 60 seconds with a 429 Too Many Requests error.
5. **Given** a valid Session Token, **When** the client accesses a protected endpoint (e.g., /info), **Then** the request succeeds.
6. **Given** a missing or invalid Session Token, **When** the client accesses a protected endpoint, **Then** the request is rejected (401 Unauthorized).

---

### User Story 2 - Device Inventory & Network Query (Priority: P2)

A bootstrap operator needs to query the device's hardware identity and current network state to verify compatibility and troubleshoot connectivity before attempting configuration.

**Why this priority**: Identification is critical to ensure the operator is provisioning the correct device. Network visibility helps diagnose why a device might not be reaching the management server.

**Independent Test**: Can be fully tested by authenticating and querying /info and /network endpoints.

**Acceptance Scenarios**:

1. **Given** an authenticated session, **When** the operator requests /info, **Then** the system returns TPM details (manufacturer, model, serial number), Board details (manufacturer, model, serial number), CPU architecture, OS distribution and version, and FIPS mode status.
2. **Given** an authenticated session, **When** the operator requests /network, **Then** the system returns a list of all interfaces, their link states (up/down), MAC addresses, and current IP addresses.
3. **Given** a device with no TPM, **When** /info is requested, **Then** TPM fields are returned as null without error.

---

### User Story 3 - Atomic Configuration Provisioning (Priority: P3)

A bootstrap operator needs to upload a complete set of configuration files and have them applied to `/etc` in a single, safe transaction to provision the device.

**Why this priority**: This is the core value proposition of the tool. Using a single "bundle" approach (vs. individual file uploads) ensures the device doesn't end up in a half-configured state if the connection drops.

**Independent Test**: Can be fully tested by posting a JSON configuration bundle and verifying files land in `/etc` (or a test directory) with correct permissions.

**Acceptance Scenarios**:

1. **Given** an authenticated session and a valid configuration bundle (JSON containing paths, content, and modes) where all paths are within allowed subdirectories, **When** POST /configure is called, **Then** the server writes the files to a temporary location, validates them, and moves them to their target paths in `/etc`.
2. **Given** a configuration bundle where one file path is outside the allow-list (e.g., `/etc/passwd`) or unwritable, **When** the request is processed, **Then** the entire operation fails, no files are modified in `/etc`, and a specific error is returned.
3. **Given** a bundle containing sensitive data, **When** the operation is logged to journald, **Then** the file contents are strictly redacted from the logs.

---

### User Story 4 - System Command Execution (Priority: P4)

A bootstrap operator needs to execute specific lifecycle commands (e.g., reboot, reload services) to activate the applied configurations.

**Why this priority**: Configuration files in `/etc` are useless until the relevant services reload or the system reboots.

**Independent Test**: Can be tested by triggering allowed commands and observing system behavior (service reload or mock script execution).

**Acceptance Scenarios**:

1. **Given** an authenticated session, **When** the operator requests POST /command with an allowed ID (e.g., "reboot"), **Then** the command is executed via sudo and the output/exit code is returned.
2. **Given** a request for a command NOT in the allow-list (e.g., "rm -rf /"), **When** processed, **Then** the request is rejected immediately.
3. **Given** a command that produces output, **When** executed, **Then** the stdout and stderr are captured and returned in the JSON response.

---

### User Story 5 - Automatic Lifecycle Management (Priority: P5)

The system needs to manage its own lifecycle to ensure it remains ephemeral (does not run on provisioned devices) and resource-efficient (exits when idle). The operator explicitly signals provisioning completion via an API endpoint.

**Why this priority**: adheres to the "Inert" and "Ephemeral" constitution principles. Prevents the tool from becoming a security liability on production devices.

**Independent Test**: Can be tested by creating the sentinel file and attempting to start the service, calling POST /complete, or by waiting for the timeout.

**Acceptance Scenarios**:

1. **Given** the service is starting up, **When** the sentinel file (`/etc/boardingpass/issued`) exists, **Then** the process exits immediately with code 0.
2. **Given** a running service, **When** no API requests are received for 10 minutes (default), **Then** the service self-terminates.
3. **Given** an authenticated session after successful provisioning, **When** the operator requests POST /complete, **Then** the sentinel file is created and the service begins graceful termination.
4. **Given** the POST /complete endpoint has been called, **When** subsequent API requests are attempted, **Then** they may fail as the service shuts down.

---

### Edge Cases

- **Brute Force Attacks**: Failed SRP authentication attempts trigger progressive delays (1s after first failure, 2s after second, 5s after third, then 60s lockout), balancing security with legitimate user retry needs.
- **Clock Skew**: How does the system handle Session Token expiration (30-minute lifetime) if the device clock is significantly different from the client clock?
- **Disk Full**: What happens if the filesystem fills up during the "write to temp" phase of configuration provisioning?
- **Power Loss**: What happens if power is cut exactly when moving files from temp to `/etc`? (Filesystem dependent, but likely atomic rename).
- **Verifier Missing**: What happens if the service starts but cannot find the SRP verifier/salt file?

## Requirements *(mandatory)*

### Functional Requirements

#### API & Security

- **FR-001**: System MUST expose a RESTful API over HTTPS (TLS 1.3 or higher).
- **FR-001a**: System MUST use a TLS certificate provided in the OS image if present; otherwise, generate a self-signed certificate at first boot and persist it for the service lifetime.
- **FR-002**: System MUST implement SRP-6a (Secure Remote Password) mutual authentication.
- **FR-002a**: System MUST implement progressive delay after failed authentication attempts: 1 second after first failure, 2 seconds after second failure, 5 seconds after third failure, then reject all authentication attempts for 60 seconds.
- **FR-003**: System MUST NOT accept any API requests (except auth handshake) without a valid Session Token.
- **FR-003a**: System MUST reject Session Tokens that have expired (30 minutes from issuance) with a 401 Unauthorized response.
- **FR-004**: System MUST read the SRP Salt and Verifier from a secured, read-only local source (e.g., `/etc/boardingpass/verifier`).
- **FR-005**: System MUST bind to a configurable port (default: 8443) on all interfaces.
- **FR-006**: System MUST log requests to journald in JSON format, strictly excluding configuration payloads, secrets, and SRP session keys.

#### Device Inventory (Info & Network)

- **FR-007**: System MUST provide an endpoint (GET /info) returning TPM, Board, CPU, and OS metadata.
- **FR-008**: System MUST provide an endpoint (GET /network) returning Interface Name, MAC, Link State, and IP Addresses.

#### Configuration Provisioning

- **FR-009**: System MUST accept a "Configuration Bundle" containing multiple file definitions in a single HTTP request.
- **FR-009a**: System MUST validate each file path in the bundle against a configurable allow-list of permitted subdirectories (e.g., `/etc/systemd/`, `/etc/myapp/`) to prevent writes to critical system files.
- **FR-010**: System MUST write bundle files to a temporary directory first, validating write success before moving to target paths.
- **FR-011**: System MUST ensure file operations are as atomic as possible (using rename syscalls).
- **FR-012**: System MUST fail the entire request if any single file in the bundle cannot be written or violates the path allow-list.

#### Command Execution

- **FR-013**: System MUST provide an endpoint (POST /command) to execute commands.
- **FR-014**: System MUST validate requested commands against a strict, pre-configured allow-list (mapping string IDs to filesystem paths).
- **FR-015**: System MUST use sudo to execute allow-listed commands.

#### Lifecycle

- **FR-016**: System MUST check for the existence of `/etc/boardingpass/issued` at startup and exit if present.
- **FR-017**: System MUST provide an endpoint (POST /complete) that creates `/etc/boardingpass/issued` and initiates graceful service termination.
- **FR-018**: System MUST terminate after a configurable timeout (default: 10m) of inactivity.

### Key Entities

- **SRP Verifier**: The cryptographic derivation of the password (v = g^x % N) stored on the device. It allows the device to verify the password without storing the password itself.

- **Session Token**: A bearer token (e.g., JWT or high-entropy string) issued after SRP success, used to authenticate subsequent REST calls. Tokens expire after 30 minutes from issuance to limit exposure if compromised while allowing sufficient time for complete provisioning workflows.

- **System Info**: Represents the immutable or rarely-changing hardware and software characteristics of the device. Includes TPM details (manufacturer, model, serial), board details (manufacturer, model, serial), CPU architecture, OS distribution and version, and FIPS mode status. This entity is read-only and derived from system inspection of static attributes.

- **Network Configuration**: Represents the list of network interfaces and their types, names, MAC addresses, link states (up/down), and assigned IP addresses (IPv4 and IPv6). This entity is read-only and reflects real-time network state at query time.

- **Configuration Bundle**: A JSON structure representing the desired state of multiple files (Path, Permissions, Base64 Content). Each file path must be validated against the configured allow-list of permitted subdirectories to prevent writes to critical system files.

- **Sentinel File**: A specific file path (`/etc/boardingpass/issued`) indicating the device is "owned" and the BoardingPass service should no longer run.

- **Service Configuration**: Represents the runtime configuration of the BoardingPass service. Includes configuration of enbled transports (initially just Ethernet, later optionally WiFi, BLE, USB) and the configuration of each transport (e.g. interfaces to bind to, well-known IP address and TCP port to listen on), inactivity timeout duration, command allow-list, configuration file path allow-list (permitted subdirectories for bundle writes), health check script path. Loaded at service startup from configuration file in YAML or JSON format.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: SRP Handshake completes in < 500ms on reference hardware (Raspberry Pi 4 or equivalent).
- **SC-002**: System consumes < 50MB RAM while idle.
- **SC-003**: Configuration bundles up to 10MB are processed and applied within 5 seconds.
- **SC-004**: 100% of requests missing a Session Token are rejected with 401 Unauthorized.
- **SC-005**: Service successfully self-terminates within 5 seconds of the inactivity timeout threshold.
- **SC-006**: Binary size (stripped) is < 10MB.

### Deliverables

As part of this feature specification, the following technical artifacts must be produced:

- **OpenAPI 3.1 Specification**: A complete definition of the API endpoints, including the SRP-6a authentication flows and schemas.
- **Binary Artifact**: A statically linked binary (`boardingpass`) compiled for the target architecture (x86_64/ARM64).
- **systemd Unit File**: A production-ready .service file handling the startup checks (sentinel file) and lifecycle.
- **sudoers Configuration**: A restrictive `/etc/sudoers.d/boardingpass` file allowing only the specific commands required.
- **SRP Tooling**: A utility or script to generate the SRP salt and verifier from a plain-text password during the OS image build process.
- **TLS Certificate Tooling**: Optional utility to pre-generate TLS certificates during OS image build; the service will auto-generate self-signed certificates at first boot if none are provided.
- **Packaging Scripts**: RPM spec file and DEB control files for distribution.

## Assumptions

- **Shared Secret Distribution**: Assumes the bootstrap operator (human) has access to the device's specific password/PIN (e.g., printed on a label or displayed on a screen) to initiate the SRP handshake.

- **TLS Certificate Trust**: Assumes the bootstrap operator client accepts self-signed certificates or is configured to skip certificate validation, since SRP provides the actual authentication security. TLS provides encryption but not certificate-based trust in this use case.

- **Verifier Persistence**: Assumes the OS image build process includes a step to generate the SRP verifier and salt and place them in the read-only location expected by the BoardingPass service.

- **Network Stack**: Assumes that the IP address the BoardingPass service listens on on a given network interface is either a "well-known address" configured at startup of the BoardingPass service based on the service's configuration file or is auto-configured (DHCP or link local) for the network interface.

- **Clock Synchronization**: Assumes the device clock is reasonably accurate or that the SRP implementation tolerates minor skew, though SRP is generally robust against clock issues unlike strict certificate validation.

- **Serialization**: Assumes configuration provisioning is a blocking operation; the system does not support simultaneous configuration uploads from multiple operators.