# Feature Specification: Mobile Device Onboarding App

**Feature Branch**: `003-mobile-onboarding-app`
**Created**: 2025-12-10
**Status**: Draft
**Input**: User description: "Build a React Native app for iOS and Android that helps users onboard headless Linux devices by interacting with the BoardingPass service through its RESTful API."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Device Discovery (Priority: P1)

As a system administrator, I need to discover nearby headless Linux devices that are ready for onboarding so that I can configure them without needing to know their IP addresses in advance.

**Why this priority**: Device discovery is the foundation of the onboarding workflow. Without the ability to find devices, no other functionality is possible. This story delivers immediate value by showing users what devices are available.

**Independent Test**: Can be fully tested by launching the app near a BoardingPass-enabled device and verifying the device appears in the list. Delivers value by providing visibility into available devices.

**Acceptance Scenarios**:

1. **Given** I open the app, **When** the app starts, **Then** I see an empty device list with a message indicating scanning is in progress
2. **Given** a BoardingPass device is broadcasting on the local network, **When** the app scans for devices, **Then** the device appears in the list showing its name and IP address
3. **Given** no devices are broadcasting on the network, **When** the app finishes scanning, **Then** I see a message indicating no devices were found
4. **Given** multiple devices are broadcasting, **When** the app scans, **Then** all devices appear in the list with unique identifiers
5. **Given** a device stops broadcasting, **When** it goes offline, **Then** it is removed from the device list
6. **Given** the automatic network scan fails to find devices, **When** I view the device list, **Then** I see devices from the well-known fallback address if available

---

### User Story 2 - Device Authentication (Priority: P2)

As a system administrator, I need to authenticate with a discovered device using a secure connection code so that I can establish a trusted connection and begin the onboarding process.

**Why this priority**: Authentication is essential for security but depends on having discovered a device first. This story enables users to establish secure connections to devices.

**Independent Test**: Can be tested by discovering a device, tapping the "+" button, and successfully authenticating with a valid connection code. Delivers value by enabling secure access to device configuration.

**Acceptance Scenarios**:

1. **Given** I see a device in the list, **When** I tap the "+" button next to it, **Then** I am prompted to enter or scan a connection code
2. **Given** I am on the connection code screen, **When** I choose to enter the code manually, **Then** I see a text input field for the connection code
3. **Given** I am on the connection code screen, **When** I choose to scan a code, **Then** the camera opens for QR code or barcode scanning
4. **Given** I scan a valid QR code or barcode, **When** the code is recognized, **Then** the connection code field is populated automatically
5. **Given** I enter or scan a connection code, **When** I submit it, **Then** the app attempts to authenticate with the device
6. **Given** authentication succeeds, **When** the connection is established, **Then** I proceed to the device information screen
7. **Given** authentication fails due to invalid code, **When** the error occurs, **Then** I see a clear error message with guidance to retry
8. **Given** authentication fails due to network issues, **When** the error occurs, **Then** I see a specific error message indicating the network problem
9. **Given** the device is unavailable during authentication, **When** the connection attempt fails, **Then** I see an actionable error message and option to return to device list

---

### User Story 3 - Device Information Display (Priority: P3)

As a system administrator, I need to view detailed information about a device after authentication so that I can verify I'm onboarding the correct device and understand its current state.

**Why this priority**: Information display builds upon authentication and provides confirmation before provisioning. While important, users can technically proceed with onboarding without viewing all details.

**Independent Test**: Can be tested by authenticating with a device and verifying that device information and network configuration are displayed correctly. Delivers value by providing visibility and confidence in the onboarding process.

**Acceptance Scenarios**:

1. **Given** I successfully authenticate with a device, **When** authentication completes, **Then** the app queries the device for system information
2. **Given** the app queries device information, **When** the data is retrieved, **Then** I see details including device name, hardware identifiers, and system specifications
3. **Given** I am viewing device information, **When** the screen loads, **Then** I also see network configuration details including interface names, link states, and IP addresses
4. **Given** the device information query fails, **When** the error occurs, **Then** I see an error message with option to retry or return to device list
5. **Given** I am viewing device information, **When** I need to proceed with onboarding, **Then** I see clear next steps or actions available
6. **Given** partial data is available, **When** some queries succeed and others fail, **Then** I see the available data with indicators for missing information

---

### User Story 4 - Error Recovery and User Guidance (Priority: P2)

As a system administrator, I need clear error messages and recovery options when things go wrong so that I can troubleshoot issues and successfully complete the onboarding process.

**Why this priority**: Error handling is critical for production use. Without proper error recovery, users will get stuck and frustrated, making the app unusable in real-world scenarios.

**Independent Test**: Can be tested by simulating various failure scenarios (network disconnection, invalid codes, device unavailability) and verifying appropriate error messages and recovery paths are presented.

**Acceptance Scenarios**:

1. **Given** a network error occurs during device discovery, **When** the scan fails, **Then** I see an error message explaining the issue with an option to retry
2. **Given** the camera permission is denied, **When** I try to scan a QR code, **Then** I see a message explaining why permission is needed with a link to settings
3. **Given** an authentication request times out, **When** the timeout occurs, **Then** I see a message indicating the timeout with options to retry or use a different device
4. **Given** the device becomes unreachable after authentication, **When** information queries fail, **Then** I see clear guidance about what happened and how to proceed
5. **Given** any error state occurs, **When** I view the error message, **Then** the message is in plain language without technical jargon

---

### Edge Cases

- What happens when the device list updates while the user is viewing it (devices appear/disappear)?
- How does the app handle devices with identical names on the network?
- What happens if the user backgrounds the app during authentication?
- How does the app handle very slow network connections or partial connectivity?
- What happens when a QR code scan produces invalid or malformed data?
- How does the app handle devices running different versions of BoardingPass with potentially different API schemas?
- What happens when the user denies camera permissions after previously granting them?
- How does the app behave when switching between WiFi and cellular networks during operation?
- What happens if a device broadcasts with incomplete or malformed mDNS data?

## Requirements *(mandatory)*

### Functional Requirements

#### Device Discovery

- **FR-001**: App MUST scan for devices broadcasting via mDNS service name `_boardingpass._tcp`
- **FR-002**: App MUST attempt connection to well-known fallback IP address (192.168.1.100:9443) when network scanning is available
- **FR-003**: App MUST display discovered devices in a list showing device name and IP address
- **FR-004**: App MUST provide a visual indicator (button) to initiate onboarding for each discovered device
- **FR-005**: App MUST automatically refresh the device list when devices appear or disappear from the network
- **FR-006**: App MUST handle duplicate device names by displaying additional identifying information

#### Authentication

- **FR-007**: App MUST provide option to enter connection code via text input
- **FR-008**: App MUST provide option to scan connection code via QR code or barcode
- **FR-009**: App MUST use camera to scan QR codes and barcodes when user selects scan option
- **FR-010**: App MUST authenticate with BoardingPass device using the `/auth/srp` endpoint
- **FR-011**: App MUST handle SRP-6a authentication protocol correctly
- **FR-012**: App MUST securely store session tokens after successful authentication
- **FR-013**: App MUST clear sensitive authentication data when authentication fails or session ends

#### Device Information Retrieval

- **FR-014**: App MUST query device information from `/info` endpoint after successful authentication
- **FR-015**: App MUST query network configuration from `/network` endpoint after successful authentication
- **FR-016**: App MUST display system information including hardware identifiers and specifications
- **FR-017**: App MUST display network configuration including interface details, link states, and IP addresses
- **FR-018**: App MUST format and present device data in a user-friendly, readable manner

#### User Experience

- **FR-019**: App MUST provide clear visual feedback for all loading states (scanning, authenticating, querying)
- **FR-020**: App MUST use intuitive navigation patterns consistent with platform conventions (iOS/Android)
- **FR-021**: App MUST follow Material Design principles via React Native Paper components
- **FR-022**: App MUST be responsive and performant on both iOS and Android devices
- **FR-023**: App MUST handle screen orientation changes gracefully

#### Error Handling

- **FR-024**: App MUST display clear, actionable error messages for all failure scenarios
- **FR-025**: App MUST provide retry mechanisms for transient failures (network issues, timeouts)
- **FR-026**: App MUST handle permission denials (camera) with clear guidance to user
- **FR-027**: App MUST validate all user inputs before submission
- **FR-028**: App MUST handle API errors gracefully with appropriate user feedback
- **FR-029**: App MUST log errors for debugging while never displaying sensitive data to users

#### Security

- **FR-030**: App MUST use HTTPS for all communication with BoardingPass devices
- **FR-031**: App MUST accept self-signed certificates with certificate pinning after first successful connection
- **FR-032**: App MUST display certificate status indicator showing whether certificate is self-signed or signed by a trusted CA
- **FR-033**: App MUST provide certificate information summary (issuer, subject, validity period, fingerprint) accessible via tooltip or popup
- **FR-034**: App MUST prompt user for explicit trust confirmation on first connection to device with self-signed certificate
- **FR-035**: App MUST validate pinned certificates on subsequent connections and alert user if certificate changes
- **FR-036**: App MUST NOT store connection codes persistently
- **FR-037**: App MUST implement secure session management with appropriate timeouts
- **FR-038**: App MUST handle authentication failures with rate limiting or progressive delays to prevent brute force attacks

### Key Entities

- **Device**: Represents a headless Linux device running BoardingPass service
  - Attributes: name, IP address, port, mDNS service info, online status, hardware identifiers
  - Relationships: has network interfaces, has system information, has authentication session

- **Authentication Session**: Represents a secure connection to a device
  - Attributes: session token, expiration time, device reference, authentication state
  - Relationships: belongs to one device, created from connection code

- **Connection Code**: Represents authentication credential for a device
  - Attributes: code value, format (manual entry vs QR/barcode), validation state
  - Relationships: used to create authentication session

- **Device Information**: System details retrieved from authenticated device
  - Attributes: TPM info, board details, CPU info, OS info, FIPS status
  - Relationships: belongs to one device

- **Network Configuration**: Network details for authenticated device
  - Attributes: interface name, link state, IP addresses, MAC address
  - Relationships: belongs to one device, multiple interfaces per device

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can discover nearby devices within 10 seconds of app launch
- **SC-002**: Users can complete authentication with a valid connection code in under 30 seconds
- **SC-003**: Users can view complete device information within 5 seconds of successful authentication
- **SC-004**: 95% of users successfully onboard a device on their first attempt without external assistance
- **SC-005**: App maintains responsive UI with no perceived lag during normal operations
- **SC-006**: App successfully handles network disruptions with clear error messages and recovery options
- **SC-007**: QR code scanning succeeds in under 3 seconds with proper lighting
- **SC-008**: App supports onboarding workflow on both iOS and Android with consistent behavior
- **SC-009**: Zero critical security vulnerabilities in authentication flow (validated via security audit)
- **SC-010**: Users report high satisfaction (4+ out of 5) with app usability and clarity of error messages

## Assumptions *(include if making assumptions about unclear requirements)*

### Technical Assumptions

- BoardingPass devices will consistently broadcast mDNS with service name `_boardingpass._tcp`
- BoardingPass API endpoints (`/auth/srp`, `/info`, `/network`) follow the existing OpenAPI specification in `specs/001-boardingpass-api/contracts/openapi.yaml`
- Connection codes will be provided to users through an out-of-band mechanism (printed label, separate secure channel)
- Default well-known IP address is 192.168.1.100:9443 but may be configurable

### User Experience Assumptions

- Users will have basic familiarity with mobile apps and QR code scanning
- Users will operate the app on devices with camera access for QR scanning
- Users will onboard devices on the same local network where mDNS functions properly
- Users need minimal guidance beyond clear in-app messaging

### Operational Assumptions

- App will be distributed via standard app stores (Apple App Store, Google Play)
- App will support currently supported versions of iOS and Android (last 2-3 major versions)
- Monorepo structure includes both React Native app and BoardingPass Go service code
- Development will use Expo managed workflow for cross-platform compatibility

### Security Assumptions

- BoardingPass devices may use either self-signed TLS certificates or CA-signed certificates
- App uses certificate pinning strategy to protect against MITM attacks after initial trust establishment
- Users will review and accept certificate information on first connection to devices with self-signed certificates
- Connection codes are sufficiently strong and randomly generated
- Session tokens from BoardingPass service have appropriate expiration (30 minutes per existing spec)
- Users will not share connection codes across insecure channels

## Dependencies *(include if feature relies on external factors)*

### External Dependencies

- **BoardingPass Service API**: App depends on BoardingPass service running on target devices with functional `/auth/srp`, `/info`, and `/network` endpoints
- **Network Infrastructure**: mDNS requires properly configured local network that allows multicast traffic
- **Device Capabilities**: Requires camera hardware and permissions for QR/barcode scanning
- **Platform Permissions**: Requires camera permissions (iOS/Android), network access permissions

### Technical Dependencies

- **React Native**: Core framework for cross-platform mobile development
- **Expo**: Development and build toolchain
- **React Native Paper**: Material Design component library
- **react-native-zeroconf**: mDNS/Bonjour service discovery library
- **react-native-vision-camera**: Camera access for QR/barcode scanning
- **Axios**: HTTP client for API communication
- **TypeScript**: Type safety and developer experience

### Integration Dependencies

- **OpenAPI Specification**: App must remain compatible with BoardingPass API as defined in `specs/001-boardingpass-api/contracts/openapi.yaml`
- **SRP-6a Protocol**: Authentication requires correct implementation of SRP-6a protocol matching BoardingPass service
- **Protocol Types**: Shared types from `pkg/protocol/` package for API request/response structures

## Out of Scope *(include to clearly define boundaries)*

The following features are explicitly excluded from this initial version:

- **Device Provisioning**: Uploading configuration files or executing commands on devices (future feature)
- **Multi-device Management**: Managing multiple onboarded devices simultaneously
- **Device History**: Tracking previously onboarded devices or onboarding history
- **Offline Mode**: Operating without network connectivity
- **Configuration Backup/Restore**: Saving or restoring device configurations
- **User Authentication**: App-level user accounts or authentication (all auth is device-specific)
- **Device Monitoring**: Ongoing monitoring or alerts for onboarded devices
- **Custom Network Settings**: Manually configuring network settings within the app
- **Device Firmware Updates**: Updating BoardingPass service or device firmware
- **Detailed Logging/Diagnostics**: Advanced troubleshooting or log export features
- **Multiple Connection Methods**: Alternative authentication methods beyond SRP-6a
- **Device Grouping/Organization**: Organizing devices into groups or categories
- **Search/Filter**: Searching or filtering device list (relevant when list is small)

## Risks *(include if significant risks exist)*

### Technical Risks

- **mDNS Reliability**: Corporate or restricted networks may block mDNS multicast, making device discovery fail
  - Mitigation: Fallback to well-known IP addresses, provide manual IP entry option in future iteration

- **TLS Certificate Validation**: Self-signed certificates from BoardingPass require user trust decision on first connection
  - Mitigation: Certificate pinning after initial trust, clear certificate information display with status indicators, user alerts on certificate changes

- **Cross-Platform Inconsistencies**: React Native libraries (especially camera and mDNS) may behave differently on iOS vs Android
  - Mitigation: Extensive testing on both platforms, platform-specific adaptations where necessary

- **Session Management Complexity**: SRP-6a protocol implementation in mobile app may be complex and error-prone
  - Mitigation: Thorough testing, potential use of existing SRP libraries if available, security review

### User Experience Risks

- **QR Code Scanning Failures**: Poor lighting, damaged codes, or camera quality issues may prevent scanning
  - Mitigation: Always provide manual entry option, clear guidance on scanning requirements

- **Network Confusion**: Users may not understand network requirements or why devices aren't discovered
  - Mitigation: Clear error messages, troubleshooting guidance, network status indicators

- **Permission Friction**: Users may deny camera permissions, blocking QR scanning
  - Mitigation: Clear permission rationale, graceful degradation to manual entry

### Security Risks

- **Man-in-the-Middle Attacks**: If TLS validation is weak, attackers could intercept authentication
  - Mitigation: Proper certificate validation strategy, consider certificate pinning

- **Connection Code Exposure**: Users may inadvertently expose connection codes
  - Mitigation: Clear guidance on code security, short-lived codes, secure code generation

- **Session Hijacking**: If session tokens are compromised, unauthorized access could occur
  - Mitigation: Secure token storage, appropriate session timeouts, token refresh mechanisms
