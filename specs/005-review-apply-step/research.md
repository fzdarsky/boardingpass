# Research: Review & Apply Step

**Feature**: 005-review-apply-step
**Date**: 2026-03-05

## R1: Clock Synchronization Detection on Linux

### Decision
Use `timedatectl show` to detect clock synchronization status and current system time.

### Rationale
- RHEL 9+ always has systemd with timedatectl available
- `timedatectl show --property=NTPSynchronized --value` returns `yes` or `no` — simple, parseable
- `timedatectl show --property=TimeUSec --value` returns the current system time
- Works regardless of NTP provider (chronyd, systemd-timesyncd)
- No additional dependencies required — aligns with constitution's minimal dependency principle

### Alternatives Considered
- **`chronyc tracking`**: Provides more detail (stratum, offset, frequency) but requires chrony specifically. On RHEL 9 chronyd is default but timedatectl is more universal. The extra detail is not needed — we only need sync/not-sync status.
- **Reading `/sys/` or `/proc/` files**: No standard procfs/sysfs interface for NTP sync status. Would need to parse adjtimex(2) return value, which is fragile.
- **`ntpstat` command**: Not installed by default on RHEL 9.

## R2: Immediate Mode Sequential Execution Strategy

### Decision
Execute actions sequentially in a single async function that iterates through the action list, updating per-action status in React state after each action completes.

### Rationale
- Sequential execution is simpler than parallel and matches user expectations (seeing actions complete one by one)
- The existing `applyStepImmediate` pattern already handles sequential config+command execution per step — this extends it to handle all steps in one pass
- React state updates between actions allow real-time UI feedback (spinner → checkmark/error)
- On failure: stop iteration, mark remaining actions as pending, let user retry or go back

### Alternatives Considered
- **Parallel execution**: Would complete faster but makes error handling complex (partial failures, rollback), and the action list has sequential dependencies (e.g., network config must be applied before connectivity test). Rejected.
- **Server-side orchestration**: Would require a new endpoint on the service to accept a "plan" and execute it. Over-engineered for this use case. Rejected.

## R3: Action List Generation Approach

### Decision
Implement `buildActionList(state: WizardState): PlannedAction[]` as a pure function in a dedicated module (`action-list.ts`). Each action carries a description string, a type tag, and enough metadata to execute it.

### Rationale
- Pure function = trivially testable with table-driven tests (no mocks, no context)
- Separate module keeps the review component clean and the generation logic reusable
- Action metadata (config files, commands, check parameters) is derived from the same wizard state that `buildStepConfigFiles` and `buildStepCommands` already use — but restructured as a flat list instead of per-step groups
- The description is human-readable text generated from state values (e.g., interpolating hostname, SSID, IP addresses)

### Alternatives Considered
- **Inline in ReviewPage component**: Would mix generation logic with rendering. Harder to test. Rejected.
- **Reuse existing `buildStepConfigFiles`/`buildStepCommands`**: These return raw config/command data, not human-readable actions. The action list needs a higher-level abstraction that includes descriptions and verification steps (connectivity, DNS, clock sync) that don't exist in the current per-step builders. However, the execution phase will still delegate to these builders for the actual API calls.

## R4: Hostname Exception Handling

### Decision
Keep hostname applied immediately at Step 1 (unchanged from current behavior). The review action list shows hostname as the first action but marks it as "already applied" if it was changed, or "keep as-is" if unchanged.

### Rationale
- Hostname changes are safe (no connectivity impact) and immediate feedback is valuable
- Changing this would break existing behavior without benefit
- The review list still shows the hostname action for completeness — the user should see the full picture

### Alternatives Considered
- **Defer hostname to review step too**: Would unify the flow but delays feedback for a safe operation. Not worth the consistency gain.

## R5: "Wait for Clock Sync" Action Behavior

### Decision
In immediate mode, the "Wait for clock to be synchronised" action polls the `/info` endpoint for clock sync status, with a timeout of 30 seconds and 3-second polling interval. In deferred mode, this action is shown as informational only (described as "Wait for clock to be synchronised after reboot").

### Rationale
- After NTP configuration is applied (chronyd restart or automatic time servers), sync typically completes within 5-15 seconds
- 30-second timeout is generous enough for most cases without blocking the user excessively
- Polling `/info` reuses existing infrastructure — no new endpoint needed
- In deferred mode, the device hasn't rebooted yet, so checking sync is not possible — informational display is appropriate

### Alternatives Considered
- **Skip clock sync check entirely**: Would lose a valuable verification step. The user wants to know the clock is correct before enrollment. Rejected.
- **Add a new `/clock` endpoint**: Over-engineering. The existing `/info` endpoint will include clock data after this feature. Rejected.
- **Longer timeout (60s+)**: Blocks the user too long. 30s is sufficient for NTP sync. If it doesn't sync, the action reports the status and the user can decide.

## R6: Step Count Change (5 → 6 Steps)

### Decision
Increase `TOTAL_STEPS` from 5 to 6. Add `REVIEW: 6` to `WIZARD_STEPS` constant. Update `StepIndicator` to show 6 steps with the last labeled "Review".

### Rationale
- The review step is a real navigation destination, not a modal overlay
- Using the existing step infrastructure means back-navigation, step indicator, and keyboard-avoiding behavior all work automatically
- Step 6 follows naturally after Step 5 (Enrollment)

### Alternatives Considered
- **Modal overlay on Step 5**: Would avoid changing step count but loses step indicator integration and back-navigation consistency. Rejected.
- **Replace Step 5 content after validation**: Would lose the ability to navigate back to enrollment from review. Rejected.
