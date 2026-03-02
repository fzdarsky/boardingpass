# Feature Specification: Enrollment Configuration Wizard

**Feature Branch**: `004-enrollment-flow`
**Created**: 2026-03-02
**Status**: Draft
**Input**: User description: "Add the functionality to the BoardingPass App that allows users to configure a Linux host running the BoardingPass service in a wizard-style step-by-step manner: Hostname, Network Interface, Network Addressing, Network Services, and Enrollment Server — with smart apply behavior depending on whether the enrollment interface matches the service interface."

## User Scenarios & Testing *(mandatory)*

### User Story 1 – Device Configuration Wizard (Priority: P1)

A technician discovers and authenticates with a headless Linux device via the BoardingPass mobile app. After viewing device information, they tap "Configure" to begin a step-by-step wizard that guides them through five sequential steps:

1. **Hostname** – View and change the device's hostname
2. **Network Interface** – Choose the network interface for enrollment from a table of available interfaces
3. **Network Addressing** – Configure IPv4 and IPv6 settings for the chosen interface
4. **Network Services** – Configure NTP servers and optional HTTP proxy
5. **Enrollment Server** – Optionally register with Red Hat Insights and/or Flight Control

Each step pre-populates with the device's current configuration. The wizard enforces sequential forward/backward navigation — users cannot skip steps. Each step validates input before allowing forward progression.

**Why this priority**: This is the core feature. Without the wizard, device enrollment configuration requires manual CLI access to the headless device, which defeats the purpose of the mobile app.

**Independent Test**: A user can navigate through all 5 wizard steps, enter valid configuration at each step, and reach the apply/review stage. Can be tested with a mock device providing current configuration data.

**Acceptance Scenarios**:

1. **Given** an authenticated device, **When** the user taps "Configure", **Then** the wizard opens at Step 1 (Hostname) with the current hostname pre-populated.
2. **Given** the user is on any wizard step with valid input, **When** they tap "Next", **Then** the wizard advances to the next step.
3. **Given** the user is on any wizard step with invalid input, **When** they tap "Next", **Then** validation errors are shown inline and forward navigation is blocked.
4. **Given** the user is on Step 2 or later, **When** they tap "Back", **Then** the wizard returns to the previous step with all previously entered data preserved.
5. **Given** the user is on Step 1, **When** they tap "Back" or the close button, **Then** they are returned to the device detail screen without any changes applied.

---

### User Story 2 – Smart Configuration Application (Priority: P1)

The technician's connection to the BoardingPass service uses a specific network interface on the device. When the technician selects an enrollment interface in Step 2, the app detects whether it matches the service's current interface.

**Different interface (Immediate Apply Mode)**: Each configuration step is applied to the device immediately upon completion, giving the technician real-time feedback (success/failure). If an apply fails, the technician can correct their input and retry before moving on.

**Same interface (Deferred Apply Mode)**: Applying network changes would disrupt the app's connection to the service. Instead, all changes are queued until the wizard completes. A Review page then displays all pending changes. On user confirmation, the entire configuration is sent atomically and the device applies the changes and reboots.

**Why this priority**: Without same-interface protection, users would lose connectivity mid-configuration and be unable to complete the wizard. Without immediate-apply for different interfaces, users would only discover configuration errors after a full reboot cycle.

**Independent Test**: Connect to a device with multiple interfaces. Configure the non-service interface and verify immediate apply with feedback. Then configure the service interface and verify deferred apply with review page.

**Acceptance Scenarios**:

1. **Given** the user selects an enrollment interface different from the service interface, **When** they complete a wizard step, **Then** the configuration for that step is applied immediately and success/failure feedback is shown.
2. **Given** the user selects the same interface the service runs on, **When** they complete the final wizard step, **Then** a Review page displays all queued changes for confirmation.
3. **Given** the user is on the Review page, **When** they confirm, **Then** all changes are sent atomically and the device is instructed to apply the configuration and reboot.
4. **Given** a step's immediate apply fails (different-interface mode), **When** the user sees the error, **Then** they can correct their input and retry the apply.
5. **Given** the user is on the Review page, **When** they tap "Edit" or "Back", **Then** they can navigate back through steps to make corrections before reapplying.
6. **Given** the device reboots after same-interface apply, **When** the app detects the device is unreachable, **Then** the app shows a "Device is rebooting with new configuration" message and returns the user to the device list.

---

### User Story 3 – WiFi Network Selection (Priority: P2)

When the technician selects a WiFi interface in Step 2, the app requests a scan of available wireless networks from the device and presents them in a table. The technician selects an SSID and provides credentials if the network is secured.

**Why this priority**: WiFi enrollment extends the feature to devices without wired connectivity. However, most enterprise/edge devices use Ethernet for enrollment, making this an enhancement over the core wired flow.

**Independent Test**: Select a WiFi-capable interface and verify SSID scanning, selection, and password entry for a WPA2-secured network.

**Acceptance Scenarios**:

1. **Given** the user selects a WiFi interface in Step 2, **When** the interface is selected, **Then** the app scans for available SSIDs and displays them in a table with columns: SSID, Signal strength, Security (WPA2/WPA3/Open), Channel, Band, and Rate.
2. **Given** available SSIDs are displayed, **When** the user selects a secured SSID (WPA2/WPA3), **Then** a password input field appears below the table.
3. **Given** the user selects an open SSID, **When** they proceed, **Then** no password is required.
4. **Given** the WiFi scan returns no results, **When** the scan completes, **Then** an empty-state message is shown with a "Rescan" option.
5. **Given** the user has selected an SSID and entered a password, **When** they navigate backward and return, **Then** their WiFi selection and password are preserved.

---

### User Story 4 – Enrollment Server Registration (Priority: P2)

The technician can optionally register the device with Red Hat Insights and/or the Flight Control management platform in the final wizard step. Each service has its own enable toggle, endpoint configuration, and credentials.

**Why this priority**: Enrollment server registration is the ultimate goal of the configuration flow, but the preceding network configuration steps must function correctly first. A device can still be useful without external enrollment registration (e.g., for testing or manual management).

**Independent Test**: Enable Red Hat Insights enrollment, enter valid Organisation ID and Activation Key, and verify the configuration is included in the apply or review step.

**Acceptance Scenarios**:

1. **Given** the user is on Step 5, **When** they enable "Red Hat Insights", **Then** fields appear for Service Endpoint (defaulting to `https://cert-api.access.redhat.com`), Organisation ID, and Activation Key.
2. **Given** the user enables "Flight Control", **When** the toggle is on, **Then** fields appear for Service Endpoint, Username, and Password.
3. **Given** neither service is enabled, **When** the user taps "Finish", **Then** the step is valid and the wizard proceeds to apply/review (enrollment is optional).
4. **Given** a service is enabled with incomplete or invalid credentials, **When** the user taps "Finish", **Then** validation errors highlight the missing or invalid fields.
5. **Given** both services are enabled with valid credentials, **When** the wizard reaches apply/review, **Then** both enrollment configurations are included in the changes.

---

### Edge Cases

- What happens when the device has no configurable network interfaces (only loopback)? → The wizard should display an error at Step 2 explaining no configurable interfaces are available.
- What happens if the device's network state changes mid-wizard (e.g., cable unplugged)? → Steps relying on live data (interface list, WiFi scan) should show a warning and offer a refresh option.
- What happens if a WiFi scan returns no results? → Show an empty-state with a "Rescan" button and the option to enter SSID manually.
- What happens if the deferred-apply reboot takes too long or fails? → After a timeout, the app shows a message that the device may need manual attention and returns to the device list.
- What happens if immediate-apply succeeds for some steps but fails for a later step? → Previously applied steps remain applied. The user can correct and retry the failed step, or navigate back to change prior steps.
- What happens if the user's session token expires during the wizard? → The wizard should prompt re-authentication and resume at the current step.
- What happens if the enrollment server endpoint is unreachable during configuration? → Endpoint reachability is not validated during the wizard — only format validation is performed. Actual connectivity is verified during apply.
- How does the app handle the device becoming unreachable after same-interface reboot? → The app shows a completion message ("Device is rebooting with new configuration"), waits briefly, and returns to the device list. The user must re-discover the device at its new address if the IP changed.

## Requirements *(mandatory)*

### Functional Requirements

**Wizard Navigation & UX**

- **FR-001**: The wizard MUST present exactly 5 sequential steps: Hostname, Network Interface, Network Addressing, Network Services, and Enrollment Server.
- **FR-002**: Each wizard step MUST pre-populate input fields with the device's current configuration when available.
- **FR-003**: The wizard MUST validate all input on the current step before allowing forward navigation to the next step.
- **FR-004**: The wizard MUST allow backward navigation to any previously completed step, preserving all entered data.
- **FR-005**: The wizard MUST NOT allow skipping steps — navigation is limited to sequential "Next" and "Back" buttons.
- **FR-006**: The wizard MUST display clear validation error messages adjacent to the invalid fields.
- **FR-007**: The wizard MUST display a step indicator showing the current step, total steps, and completion progress.
- **FR-008**: The wizard state MUST be ephemeral — closing the wizard discards all unsaved configuration.

**Step 1 – Hostname**

- **FR-010**: The hostname field MUST default to the device's current hostname, retrieved from the device.
- **FR-011**: The hostname MUST be validated per RFC 1123: alphanumeric characters and hyphens only, 1–63 characters per label, up to 253 characters total, must not start or end with a hyphen.

**Step 2 – Network Interface Selection**

- **FR-020**: The app MUST display all non-loopback network interfaces in a table with columns: Name, Type (Ethernet, WiFi, etc.), MAC Address, Vendor and Model, Speed, and State (up/down, carrier).
- **FR-021**: The user MUST select exactly one interface for enrollment.
- **FR-022**: The app MUST visually indicate which interface the BoardingPass service is currently using (the "service interface").
- **FR-023**: If no configurable interfaces are available, the app MUST display an error message and prevent wizard progression.
- **FR-024**: The user MAY optionally configure a VLAN ID (1–4094) for the selected interface. When specified, the device MUST tag traffic on that interface with the given VLAN ID.

**Step 2a – WiFi Network Selection (when WiFi interface selected)**

- **FR-025**: When a WiFi interface is selected, the app MUST request a scan of available wireless networks from the device.
- **FR-026**: The app MUST display available SSIDs in a table with columns: SSID, Signal Strength, Security (WPA2/WPA3/Open/etc.), Channel, Band, and Rate.
- **FR-027**: When a secured SSID is selected, the app MUST display a password input field.
- **FR-028**: When an open SSID is selected, no password MUST be required.
- **FR-029**: The app MUST offer a "Rescan" option to refresh the SSID list.

**Step 3 – Network Addressing**

- **FR-030**: For IPv4, the user MUST choose between DHCPv4 or Static IP configuration via a radio button.
- **FR-031**: If Static IPv4, the user MUST provide: IPv4 address, subnet mask, and gateway IP.
- **FR-032**: The user MUST have a "Configure DNS automatically" checkbox (default: on). When off, the user provides primary and optionally secondary DNS servers.
- **FR-033**: For IPv6, the user MUST choose between DHCPv6, Static IP, or Disabled via a radio button.
- **FR-034**: If Static IPv6, the user MUST provide: IPv6 address with prefix length (e.g., `2001:db8::1/64`) and gateway IP.
- **FR-035**: If manual IPv6 DNS is selected, the user MUST provide primary and optionally secondary IPv6 DNS servers.
- **FR-036**: All IP addresses MUST be validated for format correctness (valid IPv4 dotted-decimal or IPv6 notation).
- **FR-037**: Subnet masks MUST be validated as valid CIDR notation or dotted-decimal masks.
- **FR-038**: Gateway addresses MUST be validated as reachable within the configured subnet.

**Step 4 – Network Services**

- **FR-040**: The user MUST choose NTP configuration as either Automatic (system default) or Manual (user-provided list of NTP server hostnames or IPs).
- **FR-041**: Manual NTP servers MUST be validated as valid hostnames or IP addresses.
- **FR-042**: The user MAY configure an optional HTTP proxy with hostname and port.
- **FR-043**: The HTTP proxy MAY optionally include username and password for authenticated proxies.
- **FR-044**: Proxy hostname MUST be validated as a valid hostname or IP address; port MUST be a valid port number (1–65535).

**Step 5 – Enrollment Server**

- **FR-050**: Red Hat Insights enrollment MUST be optional (toggle on/off).
- **FR-051**: When Red Hat Insights is enabled, the Service Endpoint field MUST default to `https://cert-api.access.redhat.com`.
- **FR-052**: When Red Hat Insights is enabled, the user MUST provide Organisation ID and Activation Key.
- **FR-053**: Flight Control enrollment MUST be optional (toggle on/off).
- **FR-054**: When Flight Control is enabled, the user MUST provide Service Endpoint, Username, and Password.
- **FR-055**: All enrollment server endpoints MUST be validated as valid HTTPS URLs.

**Apply Strategy**

- **FR-060**: The app MUST determine which network interface the BoardingPass service is currently using by comparing the app's connection target IP against the device's interface IP addresses.
- **FR-061**: If the selected enrollment interface differs from the service interface (**Immediate Apply Mode**), configuration MUST be applied step-by-step with immediate success/failure feedback after each step.
- **FR-062**: If the selected enrollment interface matches the service interface (**Deferred Apply Mode**), all configuration changes MUST be queued until the wizard completes.
- **FR-063**: In Deferred Apply Mode, a Review page MUST display all queued configuration changes in a human-readable summary before application.
- **FR-064**: In Deferred Apply Mode, the user MUST explicitly confirm before the configuration is sent to the device.
- **FR-065**: In Deferred Apply Mode, after user confirmation the device MUST apply all changes atomically and reboot.
- **FR-066**: In Immediate Apply Mode, if a step's apply fails, the user MUST be shown the error and allowed to correct input and retry before proceeding.
- **FR-067**: In Immediate Apply Mode, each apply MUST provide clear success or failure feedback within a reasonable time.
- **FR-068**: After network configuration is applied (in either mode), the device MUST perform a connectivity verification (DNS resolution and basic reachability test) and report the result to the app before the wizard proceeds.

**Security**

- **FR-070**: Sensitive credentials (WiFi passwords, proxy passwords, activation keys, enrollment passwords) MUST NOT be logged or persisted beyond the wizard session.
- **FR-071**: All configuration data MUST be transmitted over the existing authenticated and TLS-encrypted session.
- **FR-072**: Credential fields MUST use secure text input (masked characters).
- **FR-073**: Wizard state, including any credentials, MUST be cleared from memory when the wizard is closed or the session ends.

### Key Entities

- **WizardState**: The ephemeral state of the configuration wizard — current step number, collected configuration for all steps, apply mode (immediate/deferred), and per-step apply status (pending/applied/failed).
- **InterfaceInfo**: Network interface metadata — name, type (Ethernet/WiFi/other), MAC address, vendor, model, speed, link state, carrier status, and current IP addresses. Extends the existing network configuration data with additional hardware details.
- **WiFiNetwork**: Available wireless network — SSID, signal strength (as percentage or dBm), security protocol (WPA2/WPA3/Open), channel number, frequency band (2.4 GHz/5 GHz/6 GHz), and data rate.
- **AddressConfig**: Network addressing configuration per interface — IPv4 method (DHCP/Static), IPv4 address/mask/gateway, DNS auto flag, DNS servers; IPv6 method (DHCP/Static/Disabled), IPv6 address with prefix/gateway, IPv6 DNS servers.
- **EnrollmentConfig**: Enrollment server settings per service — enabled flag, service type (Insights/FlightControl), endpoint URL, and type-specific credentials (Org ID + Activation Key for Insights; Username + Password for Flight Control).
- **NetworkServicesConfig**: NTP mode (automatic/manual) with optional server list; HTTP proxy settings (hostname, port, optional username/password).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can complete the full 5-step configuration wizard in under 5 minutes for a typical Ethernet enrollment with static IP.
- **SC-002**: 90% of users successfully apply device configuration on their first wizard completion without needing to go back and correct errors.
- **SC-003**: All input validation errors are displayed inline within 1 second of the user attempting to proceed.
- **SC-004**: In Immediate Apply Mode, each step's configuration is applied and feedback is shown within 10 seconds.
- **SC-005**: In Deferred Apply Mode, the Review page accurately and completely reflects all queued changes, matching what was entered in each step.
- **SC-006**: After same-interface apply and reboot, the device comes online with the new configuration within 2 minutes.
- **SC-007**: No sensitive credentials (WiFi passwords, activation keys, enrollment passwords, proxy credentials) are logged, persisted to storage, or retained in memory after the wizard closes.

## Assumptions

- The BoardingPass service API will be extended with endpoints or commands to support querying extended interface details (type, vendor, model, speed), WiFi SSID scanning, hostname changes, network address configuration, and service configuration (NTP, proxy, enrollment).
- The device runs a network management service (e.g., NetworkManager) that supports programmatic configuration changes.
- WiFi scanning capability depends on the device having wireless hardware and the service having appropriate system permissions.
- The "apply and reboot" operation in Deferred Apply Mode leverages the service's existing command execution and lifecycle management capabilities.
- The existing `/network` endpoint's data model will be extended to include interface type, vendor/model, and speed information.
- Enrollment server registration (Insights, Flight Control) is performed by provisioning the appropriate configuration files and/or executing registration commands on the device — the mobile app does not communicate directly with enrollment servers.
