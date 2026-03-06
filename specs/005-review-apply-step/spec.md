# Feature Specification: Review & Apply Step

**Feature Branch**: `005-review-apply-step`
**Created**: 2026-03-05
**Status**: Draft
**Input**: User description: "Build a Review & Apply step for the configuration wizard with unified apply flow, system time/clock sync in /info endpoint"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Review and Apply in Immediate Mode (Priority: P1)

A technician has completed all five wizard steps (Hostname, Interface, Addressing, Services, Enrollment) using an enrollment interface that differs from the service interface (immediate mode). After the final step, they advance to a "Review & Apply" screen. This screen shows a human-readable summary of every action that will be taken — written as a numbered action list rather than a raw configuration dump. The technician reviews the actions, then taps "Apply". The app executes each action sequentially, showing a checkmark or error icon next to each action as it completes. When all actions succeed, the app signals the service to set the sentinel file and shut down (no reboot needed). The app then navigates back to the device list.

**Why this priority**: This is the primary flow for most field deployments where a separate enrollment interface is used. It replaces the current per-step "Apply & Next" behavior with a single, consolidated apply step.

**Independent Test**: Can be fully tested by navigating through all wizard steps, reaching the Review & Apply screen, and verifying that tapping "Apply" executes all actions with real-time visual feedback.

**Acceptance Scenarios**:

1. **Given** a completed wizard in immediate mode, **When** the user advances past the Enrollment step, **Then** the Review & Apply screen is shown with a numbered list of all planned actions described in human-readable language.
2. **Given** the Review & Apply screen in immediate mode, **When** the user taps "Apply", **Then** each action is executed sequentially and a checkmark or error icon appears next to each action as it completes.
3. **Given** an action fails during apply, **When** the error is displayed, **Then** the user sees an error icon and a description of the failure next to the failed action, and remaining actions are halted.
4. **Given** all actions complete successfully, **When** the final action finishes, **Then** the app calls the service's /complete endpoint (without reboot), shows a "Provisioning Complete" terminal state, and returns to the device list.
5. **Given** the Review & Apply screen, **When** the user taps "Back", **Then** the user returns to the Enrollment step to make changes.

---

### User Story 2 - Review and Apply in Deferred Mode (Priority: P1)

A technician uses the same interface for both service communication and enrollment (deferred mode). After completing all wizard steps, they see the same Review & Apply screen with the same action list — but the button reads "Apply & Reboot". When tapped, the app writes the configuration bundle atomically, signals the service to complete and reboot, and shows a "Device is Rebooting" terminal state.

**Why this priority**: Deferred mode is required when enrollment and service interfaces are the same (cannot apply network changes without losing connectivity). This is a co-equal critical path.

**Independent Test**: Can be tested by selecting the service interface as the enrollment interface, completing the wizard, and verifying the Apply & Reboot flow.

**Acceptance Scenarios**:

1. **Given** a completed wizard in deferred mode, **When** the user advances past the Enrollment step, **Then** the Review & Apply screen is shown with an "Apply & Reboot" button instead of "Apply".
2. **Given** the Review & Apply screen in deferred mode, **When** the user taps "Apply & Reboot", **Then** the configuration bundle is sent atomically, the service is signaled to complete with reboot, and the device rebooting terminal state is displayed.
3. **Given** the Review & Apply screen in deferred mode, **When** the user taps "Back", **Then** the user returns to the Enrollment step.

---

### User Story 3 - Remove Per-Step Apply in Immediate Mode (Priority: P1)

In the current wizard, each step in immediate mode shows "Apply & Next" which applies configuration changes before advancing. With this feature, the "Apply & Next" button is replaced with just "Next" in all steps. Configuration is no longer applied at the end of each step — it is deferred to the Review & Apply screen. The only exception is hostname, which is still applied immediately (safe because it does not affect connectivity).

**Why this priority**: This is a prerequisite for the Review & Apply flow — without removing per-step apply, actions would be applied twice.

**Independent Test**: Can be tested by navigating through the wizard in immediate mode and verifying that only "Next" (not "Apply & Next") appears, and no configuration is sent to the device until the Review & Apply step.

**Acceptance Scenarios**:

1. **Given** immediate mode is active, **When** the user is on any step (except Review & Apply), **Then** the forward navigation button reads "Next" (not "Apply & Next").
2. **Given** immediate mode is active, **When** the user navigates from Step 3 (Addressing) to Step 4 (Services), **Then** no API calls are made to /configure or /command — configuration is queued for the Review & Apply step.
3. **Given** immediate mode is active, **When** the user is on the hostname step and changes the hostname, **Then** the hostname is still applied immediately because it is safe and does not affect connectivity.

---

### User Story 4 - System Time and Clock Sync in Device Details (Priority: P2)

The BoardingPass service's /info endpoint is extended to include the current system time and clock synchronization status (whether NTP has synchronized the clock). This information is displayed in the Operating System section of the Device Details screen, helping the technician verify that the device's time is correct before or after enrollment.

**Why this priority**: System time is important context for enrollment but is not part of the core apply flow. It supports the "Wait for clock to be synchronised" action in the review list.

**Independent Test**: Can be tested by connecting to a device and viewing the Device Details page, verifying system time and sync status appear in the OS section.

**Acceptance Scenarios**:

1. **Given** a connected and authenticated device, **When** the user views the Device Details screen, **Then** the Operating System section shows the current system time and clock sync status.
2. **Given** the device's clock is synchronized via NTP, **When** the /info endpoint is queried, **Then** the response includes clock sync status as synchronized.
3. **Given** the device's clock is NOT synchronized, **When** the /info endpoint is queried, **Then** the response includes clock sync status as not synchronized.

---

### User Story 5 - Action List Describes Planned Changes (Priority: P2)

The Review & Apply screen generates a human-readable action list from the wizard state. Each action describes what will happen in plain language, tailored to the user's choices. Actions cover hostname, interface selection and WiFi connection, IP addressing, DNS, IPv6 configuration, connectivity checks, DNS resolution checks, time server configuration, clock sync wait, and enrollment steps.

**Why this priority**: The human-readable action list is what differentiates the review page from a raw configuration dump. It gives the technician confidence in what will happen.

**Independent Test**: Can be tested by completing the wizard with various configurations and verifying the action list accurately reflects all choices.

**Acceptance Scenarios**:

1. **Given** the hostname was changed from the DHCP-assigned value, **When** the review screen is displayed, **Then** the action list includes "Set hostname to [new-hostname]".
2. **Given** the hostname was NOT changed, **When** the review screen is displayed, **Then** the action list includes "Keep hostname as assigned by DHCP ([hostname])".
3. **Given** a WiFi interface was selected, **When** the review screen is displayed, **Then** the action list includes an action for connecting to the selected SSID.
4. **Given** static IPv4 was configured, **When** the review screen is displayed, **Then** the action list includes address, netmask, and gateway details.
5. **Given** enrollment into Insights is configured, **When** the review screen is displayed, **Then** the action list includes "Enroll into Insights".
6. **Given** automatic time servers are selected, **When** the review screen is displayed, **Then** the action list includes "Use automatic time servers" and "Wait for clock to be synchronised".

---

### Edge Cases

- What happens when the user taps "Apply" and the device loses connectivity mid-way through the action list (e.g., after network reconfiguration)?
  - In immediate mode, the action that caused the connectivity loss will time out and show an error. The user is informed that the device may need manual attention. Remaining actions are halted.
- What happens when the /complete endpoint call fails after all actions succeed?
  - The user sees an error message indicating the provisioning could not be finalized, with guidance that the device may need manual attention.
- What happens when no configuration changes were made (user accepted all defaults)?
  - The review screen shows only the actions that reflect defaults (e.g., "Keep hostname as assigned by DHCP", "Use automatic time servers") and the apply proceeds with just the connectivity checks and enrollment commands (if any).
- What happens when the user navigates back from Review & Apply, changes a setting, and returns?
  - The action list is regenerated from the current wizard state to reflect the updated configuration.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST display a "Review & Apply" screen as Step 6 after the Enrollment step in both immediate and deferred modes.
- **FR-002**: The Review & Apply screen MUST display a numbered list of planned actions in human-readable language, derived from the current wizard state.
- **FR-003**: In immediate mode, the Review & Apply screen MUST show an "Apply" button. When tapped, the app MUST execute each action sequentially, displaying a progress indicator (checkmark for success, error icon for failure) next to each action.
- **FR-004**: In deferred mode, the Review & Apply screen MUST show an "Apply & Reboot" button. When tapped, the app MUST send the configuration bundle atomically and signal the service to complete with reboot.
- **FR-005**: The Review & Apply screen MUST include a "Back" button that returns the user to the Enrollment step (Step 5).
- **FR-006**: In immediate mode, when all actions succeed, the app MUST call the /complete endpoint with reboot=false and display a "Provisioning Complete" terminal state.
- **FR-007**: In immediate mode, when an action fails, the app MUST halt execution, display the error, and allow the user to retry or go back.
- **FR-008**: The forward navigation button on Steps 1–5 MUST display "Next" (not "Apply & Next") in both apply modes. Configuration MUST NOT be applied until the Review & Apply step, except for hostname which is always applied immediately.
- **FR-009**: The action list MUST include connectivity verification actions when applicable: "Check IPv4 connectivity to [gateway/test-server]" and "Check DNS resolution for [domain]".
- **FR-010**: The action list MUST include time-related actions: "Use automatic time servers" or "Set manual NTP servers [servers]", and "Wait for clock to be synchronised".
- **FR-011**: The BoardingPass service /info endpoint MUST include system time (current UTC time) and clock synchronization status (synchronized or not) in the response.
- **FR-012**: The Device Details screen MUST display system time and clock sync status in the Operating System section.
- **FR-013**: Actions in the action list MUST adapt to the user's configuration choices (e.g., "Keep hostname as assigned by DHCP" vs "Set hostname to [hostname]"; "Assign IPv4 address via DHCP" vs "Assign IPv4 address manually: [details]").
- **FR-014**: The step indicator MUST reflect that the wizard now has 6 steps, with the Review & Apply step shown as the final step.

### Key Entities

- **Action**: A single planned configuration change or verification step. Has a description (human-readable text), a type (config, command, check, or wait), and an execution status (pending, running, success, or failed). Generated from wizard state.
- **Action List**: An ordered sequence of Actions derived from the wizard state. Represents the complete set of operations that will be performed during apply.
- **Clock Status**: The NTP synchronization state of the device, including whether the system clock is synchronized and the current system time. Returned as part of the /info endpoint response.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can review all planned configuration actions in a single screen before any changes are made to the device.
- **SC-002**: Users see real-time feedback (success or failure) for each individual action during the apply process, completing within 60 seconds for typical configurations (5–10 actions).
- **SC-003**: The apply process correctly handles failures by stopping at the failed action and informing the user, without leaving the device in an inconsistent state from partial application.
- **SC-004**: System time and clock sync status are visible on the Device Details page immediately after authentication, with no additional navigation required.
- **SC-005**: 100% of wizard state configurations produce an accurate, human-readable action list that matches the actual operations performed during apply.

## Assumptions

- Hostname continues to be applied immediately (Step 1) as it does not affect network connectivity — this is unchanged from the current behavior.
- The connectivity test (`connectivity-test` command) and DNS check remain informational actions that do not block the apply flow if they fail — they report results but do not halt subsequent actions.
- Clock synchronization status is read from the system's timedatectl/chronyc state on the service side — the specific implementation detail of which system tool to use is left to the planning phase.
- The "Wait for clock to be synchronised" action in the review list is informational in deferred mode (cannot wait — config is applied on reboot) and best-effort in immediate mode (polls for a reasonable timeout before reporting status).
- The step indicator in the wizard will show 6 steps total, with Step 6 labeled "Review".
