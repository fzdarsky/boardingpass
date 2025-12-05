<!--
Sync Impact Report:
- Version: [TEMPLATE] → 1.0.0 (Initial constitution - MINOR bump for new governance document)
- Modified principles: N/A (initial creation)
- Added sections:
  * Core Principles (6 principles defined)
  * Security Requirements
  * Development Workflow
  * Governance
- Removed sections: N/A
- Templates requiring updates:
  ✅ plan-template.md - verified alignment
  ✅ spec-template.md - verified alignment
  ✅ tasks-template.md - verified alignment
- Follow-up TODOs: None
-->

# BoardingPass Constitution

## Core Principles

### I. Frictionless Bootstrapping

BoardingPass MUST provide a frictionless, secure mechanism for bootstrapping headless Linux devices into a managed state by injecting bootstrap configuration from a connected provisioning app running on an iOS or Android mobile phone or tablet. The app MUST be simple to use from a UX perspective, minimizing user interaction and technical skills required. Any component running on the Linux device MUST be simple to install, require minimal to no configuration, and be compatible with modern enterprise Linux distributions (specifically targeting RHEL 9+ and distribution as `bootc` bootable container image). Bootstrap operations MUST complete without requiring physical display, keyboard, or network infrastructure beyond the transport medium or the provisioning app itself.

**Rationale**: Headless devices cannot rely on traditional interactive setup flows. Friction in bootstrapping directly translates to deployment delays and operational overhead.

### II. Ephemeral & Fail-Safe Operation

BoardingPass MUST be inert when not in use and fail-safe when active. The software MUST NOT persist daemon processes, system modifications, or background services when bootstrap is complete. If any operation fails, the system MUST fail to a safe state that does not compromise the device or prevent recovery attempts.

**Rationale**: Persistent background processes introduce security surface area, resource consumption, and maintenance burden. Fail-safe behavior prevents bricking devices during critical bootstrap operations.

### III. Minimal Footprint

BoardingPass MUST maintain minimal footprint in both download size and runtime resource consumption. Total package size MUST be measured in single-digit megabytes. Runtime memory consumption MUST be measured in tens of megabytes. The software MUST NOT require compilation or complex installation procedures on target devices.

**Rationale**: Headless devices often have constrained storage, memory, and CPU resources. Large downloads increase deployment time and bandwidth costs. Minimal resource usage ensures compatibility with resource-constrained devices and enables bootstrap operations even under adverse conditions.

### IV. Minimal Dependencies

BoardingPass MUST minimize mandatory dependencies. The software MUST rely only on packages guaranteed to exist in minimal base Linux distributions (e.g., busybox, systemd, basic networking tools). Optional dependencies MAY be supported for enhanced functionality but MUST NOT be required for core bootstrap operations. All dependencies MUST be justified in terms of necessity and evaluated for alternatives.

**Rationale**: Dependencies create fragility, increase attack surface, complicate deployment across distributions, and expand the testing matrix. Minimal dependencies ensure maximum portability and reliability.

### V. Transport Agnostic & Protocol First

BoardingPass MUST be transport-agnostic, supporting multiple connection methods (USB, BLE, WiFi) via a unified API contract. The Protocol definition acts as the single source of truth; neither the device nor the mobile app shall rely on implementation details of the other. Adding a new transport MUST NOT require modifying the core bootstrap logic.

**Rationale**: Connectivity varies by environment. A strict separation between Transport Layer and Application Logic allows the system to adapt to new hardware without refactoring the core state machine.

### VI. Open Source & Permissive Licensing

BoardingPass MUST be distributed as open source software with a permissive license (MIT, Apache 2.0, or BSD). All source code, documentation, and build artifacts MUST be publicly accessible. The license MUST permit commercial use, modification, and redistribution without copyleft requirements. Dependencies MUST also use permissive or compatible licenses approved by the Open Source Initiative.

**Rationale**: Open source with permissive licensing enables community contributions, commercial adoption, security audits, and builds trust. Copyleft licenses create barriers for enterprise adoption and embedded device manufacturers.

## Security Requirements

BoardingPass operates in a privileged context (device bootstrap) and MUST maintain security discipline:

- **Authentication**: All bootstrap communications MUST be authenticated. Support for pre-shared keys, certificate-based authentication, or challenge-response protocols is REQUIRED.
- **Encryption**: All sensitive data in transit MUST be encrypted. Transport layer security (TLS, DTLS, or equivalent) is REQUIRED for network transports.
- **Least Privilege**: Bootstrap operations MUST run with minimum required privileges. Temporary privilege escalation MUST be explicit, auditable, and immediately dropped after use.
- **Input Validation**: All external inputs (commands, configuration, data) MUST be validated before processing. Reject invalid inputs; never attempt to sanitize or fix malformed data.
- **Secrets Management**: Secrets (keys, credentials, tokens) MUST NEVER be logged, persisted to disk in plaintext, or transmitted without encryption. Secrets MUST NOT be retained in memory longer than necessary.
- **Audit Logging**: All security-relevant events (authentication attempts, privilege escalations, failures) MUST be logged with sufficient detail for forensic analysis.

## Development Workflow

### Code Quality

- **Testing**: Unit tests REQUIRED for all core logic. Integration tests REQUIRED for transport implementations. End-to-end tests REQUIRED for complete bootstrap workflows.
- **Code Review**: All changes MUST be reviewed by at least one maintainer before merge. Security-sensitive changes MUST be reviewed by a security-focused maintainer.
- **Linting & Formatting**: Code MUST pass automated linting (language-specific) and follow consistent formatting (enforced via automated tools).
- **Documentation**: All public APIs MUST be documented. Complex internal logic SHOULD be documented. Breaking changes MUST include migration guides.

### Versioning

- **Semantic Versioning**: BoardingPass MUST follow semantic versioning (MAJOR.MINOR.PATCH).
  - **MAJOR**: Breaking changes to bootstrap protocol, API contracts, or configuration format.
  - **MINOR**: New features, new transport support, backward-compatible enhancements.
  - **PATCH**: Bug fixes, security patches, documentation updates, non-functional improvements.
- **Deprecation Policy**: Breaking changes MUST be announced at least one MINOR version in advance. Deprecated features MUST emit warnings and remain functional for at least one MAJOR version cycle.

### Release Process

- **Changelog**: All releases MUST include a human-readable changelog documenting changes, bug fixes, and known issues.
- **Artifacts**: Releases MUST include pre-built binaries for common architectures (x86_64, ARM, ARM64) and platforms (Linux distributions).
- **Signatures**: Release artifacts MUST be cryptographically signed and signatures published alongside artifacts.

## Governance

This constitution defines the non-negotiable principles and requirements for BoardingPass. All development, design decisions, and contributions MUST align with these principles.

### Amendment Process

- Amendments to this constitution require explicit documentation and justification.
- Amendments MUST be proposed via pull request with rationale and impact analysis.
- Amendments MUST be reviewed and approved by at least two maintainers.
- Approved amendments trigger a version bump and update to the Last Amended date.
- Breaking amendments (removing or weakening principles) require MAJOR version bump.
- Additive amendments (new principles or clarifications) require MINOR version bump.
- Editorial amendments (typos, formatting, clarity) require PATCH version bump.

### Compliance

- All pull requests MUST be reviewed for constitution compliance before merge.
- Violations MUST be documented and justified in complexity tracking (see `.specify/templates/plan-template.md`).
- Unjustified violations MUST be rejected.
- Repeated violations indicate constitution misalignment and MAY trigger principle review.

### Documentation & Guidance

- Runtime development guidance is maintained in project documentation (README.md, docs/).
- Template files in `.specify/templates/` provide structure for planning, specification, and task management.
- Constitution alignment MUST be verified during planning phase (see plan-template.md "Constitution Check" section).

**Version**: 1.0.0 | **Ratified**: 2025-12-05 | **Last Amended**: 2025-12-05
