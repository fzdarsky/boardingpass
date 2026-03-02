# Quickstart: Enrollment Configuration Wizard

**Feature Branch**: `004-enrollment-flow`

## Overview

This feature adds a 5-step configuration wizard to the BoardingPass mobile app that guides users through device enrollment: Hostname → Network Interface → Addressing → Services → Enrollment Server.

The wizard has two apply modes:
- **Immediate** (enrollment interface ≠ service interface): config applied per-step with real-time feedback
- **Deferred** (enrollment interface = service interface): all config sent atomically, device reboots

## Architecture

```
Mobile App (React Native)              BoardingPass Service (Go)
┌──────────────────────┐               ┌────────────────────────┐
│ ConfigWizard Screen  │               │ Existing Endpoints     │
│  ├── HostnameStep    │──GET /info───→│  ├── /info (+hostname) │
│  ├── InterfaceStep   │──GET /network→│  ├── /network (+type…) │
│  ├── AddressingStep  │               │  ├── /configure        │
│  ├── ServicesStep    │──POST /config→│  ├── /command (+params)│
│  ├── EnrollmentStep  │──POST /cmd───→│  └── /complete (+boot) │
│  └── ReviewPage      │               ├────────────────────────┤
│                      │               │ New Scripts             │
│ WizardContext        │               │  ├── wifi-scan.sh      │
│  └── useReducer      │               │  ├── reload-conn.sh    │
└──────────────────────┘               │  ├── connectivity.sh   │
                                       │  ├── enroll-insights.sh│
                                       │  └── enroll-flightctl.sh
                                       └────────────────────────┘
```

## Development Setup

### Service Side (Go)

```bash
# Build and test the service
make build-service
make test-unit-service

# Key files to modify:
#   internal/network/interfaces.go    — add type, speed, carrier, driver detection
#   internal/inventory/hostname.go    — new file for hostname retrieval
#   internal/command/executor.go      — add param support (append after --)
#   internal/config/config.go         — add max_params to CommandDefinition
#   internal/api/handlers/command.go  — parse params from request
#   internal/api/handlers/complete.go — add reboot support
#   pkg/protocol/types.go            — update request/response types
```

### Mobile App (React Native)

```bash
# Install dependencies and generate types
make install-deps-app
make generate-app      # regenerate types from updated OpenAPI spec

# Run the app
make run-app-ios

# Key files to create:
#   mobile/app/device/configure.tsx              — wizard screen
#   mobile/src/components/ConfigWizard/*.tsx      — step components
#   mobile/src/hooks/useConfigWizard.ts           — wizard state management
#   mobile/src/services/api/configure.ts          — POST /configure wrapper
#   mobile/src/services/api/command.ts            — POST /command wrapper
#   mobile/src/services/api/complete.ts           — POST /complete wrapper
#   mobile/src/utils/network-validation.ts        — IP, subnet, hostname validation
#   mobile/src/utils/nm-connection.ts             — NM connection file generation
```

### Helper Scripts

```bash
# Scripts shipped with the BoardingPass package
# Location: build/scripts/ (installed to /usr/libexec/boardingpass/)
#
#   wifi-scan.sh           — wraps nmcli, outputs JSON
#   reload-connection.sh   — nmcli connection reload + up
#   connectivity-test.sh   — tests DNS, gateway, internet
#   enroll-insights.sh     — reads staging file, runs rhc connect
#   enroll-flightctl.sh    — reads staging file, runs flightctl login
```

## API Call Flows

### Immediate Mode (different interface)

```
Step 1: POST /configure [{path: "hostname", content: base64("my-device")}]
        POST /command   {id: "set-hostname", params: ["my-device"]}

Step 3: POST /configure [{path: "NetworkManager/system-connections/bp-enrollment.nmconnection", ...}]
        POST /command   {id: "reload-connection", params: ["bp-enrollment"]}
        POST /command   {id: "connectivity-test", params: ["eth0", "192.168.1.1"]}

Step 4: POST /configure [{path: "chrony.d/boardingpass-ntp.conf", ...},
                          {path: "profile.d/boardingpass-proxy.sh", ...}]
        POST /command   {id: "restart-chronyd"}

Step 5: POST /configure [{path: "boardingpass/staging/insights.json", ...}]
        POST /command   {id: "enroll-insights"}

Done:   POST /complete  {}
```

### Deferred Mode (same interface)

```
Review: POST /configure [
          {path: "hostname", ...},
          {path: "NetworkManager/system-connections/bp-enrollment.nmconnection", ...},
          {path: "chrony.d/boardingpass-ntp.conf", ...},
          {path: "profile.d/boardingpass-proxy.sh", ...},
          {path: "boardingpass/staging/insights.json", ...},
          {path: "systemd/system/boardingpass-enroll.service", ...},
          {path: "systemd/system/multi-user.target.wants/boardingpass-enroll.service", ...}
        ]

Done:   POST /complete  {"reboot": true}
```

## Testing

```bash
# Service tests
make test-unit-service         # unit tests (fast)
make test-service              # all service tests

# App tests
make test-unit-app             # unit tests (fast)
make test-app                  # all app tests

# Lint everything
make lint-all

# Key test areas:
#   - Network interface type detection (sysfs mocking)
#   - Command param validation (max_params, --, empty params)
#   - NM connection file generation (all IPv4/IPv6 combinations)
#   - Wizard step validation (each input field)
#   - Apply mode detection (same vs different interface)
#   - Deferred mode atomic bundle construction
```
