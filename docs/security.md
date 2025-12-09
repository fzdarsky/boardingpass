# BoardingPass Security Guide

**Version**: 0.1.0
**Last Updated**: 2025-12-09

## Overview

BoardingPass is designed with security as a foundational principle. This document outlines the security features, threat model, and best practices for deploying and operating BoardingPass.

---

## Security Model

### Design Principles

1. **Ephemeral Operation**: Service terminates after provisioning; no persistent attack surface
2. **Minimal Dependencies**: Go stdlib only (except gopkg.in/yaml.v3); reduces supply chain risk
3. **FIPS 140-3 Compliance**: Go stdlib crypto/* for all cryptographic operations
4. **Fail-Safe**: Atomic operations with automatic rollback on failure
5. **Defense in Depth**: Multiple layers of security controls

### Threat Model

**In Scope**:
- Unauthorized device access during bootstrap window
- Brute-force attacks against authentication
- Configuration tampering or malicious config injection
- Command injection via allow-listed commands
- Session hijacking or token theft
- Information disclosure via logs
- Denial of service during provisioning

**Out of Scope**:
- Physical attacks on device hardware
- Compromise of the OS image before deployment
- Network-level attacks (ARP spoofing, MITM on physical layer)
- Attacks after successful provisioning (service is terminated)

---

## Authentication & Authorization

### SRP-6a Protocol

BoardingPass uses SRP-6a (Secure Remote Password) for mutual authentication:

**Security Properties**:
- **Zero-knowledge proof**: Password never transmitted over the network
- **Mutual authentication**: Both client and server verify each other's identity
- **Perfect forward secrecy**: Compromising long-term credentials doesn't compromise past sessions
- **Resistance to offline dictionary attacks**: Attacker cannot brute-force password offline

**Implementation**:
- **Group**: RFC 5054 2048-bit safe prime
- **Hash function**: SHA-256 (FIPS-approved)
- **Multiplier**: k = H(N | g) per RFC 5054
- **Constant-time comparison**: All proof verifications use `crypto/subtle.ConstantTimeCompare`

### Device-Unique Passwords

Each device generates a unique password from hardware identifiers:

**Common Sources**:
- Board serial number (DMI)
- TPM endorsement key
- Network MAC address
- Combination of multiple sources

**Security Considerations**:
- Passwords should be high-entropy (12+ characters)
- Printed on device label for operator access
- Never stored in plaintext (only verifier is computed)
- Salt is unique per bootc image build (prevents rainbow tables)

**Example Password Strength**:
```
Serial number: "ABC123XYZ789" (12 chars, alphanumeric)
Entropy: ~62 bits (assuming 62-char alphabet)

Combined: "ABC123XYZ789-dc:a6:32:12:34:56" (31 chars)
Entropy: ~185 bits
```

### Session Tokens

**Format**: `<token_id>.<signature>`
- `token_id`: 32 bytes random (crypto/rand)
- `signature`: HMAC-SHA256(token_id + username, secret_key)

**Properties**:
- 30-minute TTL (configurable, minimum 5 minutes)
- In-memory storage only (not persisted)
- HMAC signature prevents tampering
- Invalidated on service restart

**Best Practices**:
- Use HTTPS only (TLS 1.3)
- Do not log session tokens
- Limit concurrent sessions (default: 10)

### Rate Limiting

Progressive delay rate limiting per client IP:

1. **1st failure**: 1-second delay
2. **2nd failure**: 2-second delay
3. **3rd failure**: 5-second delay
4. **4th+ failures**: 60-second lockout

**Effectiveness**:
- Allows legitimate operators to retry (typos)
- Prevents automated brute-force attacks
- No permanent lockout (resets after 60 seconds of inactivity)

**Brute-Force Attack Analysis**:
```
Attempts per hour (unlimited): 3600
Attempts per hour (with delays): ~65
Time to try 1M passwords: 428 hours (~18 days)
Time to try 1B passwords: 428,000 hours (~49 years)
```

---

## Transport Security

### TLS Configuration

**Requirements**:
- TLS 1.3 minimum (rejects TLS 1.2 and below)
- FIPS-approved cipher suites:
  - TLS_AES_128_GCM_SHA256
  - TLS_AES_256_GCM_SHA384
- FIPS-approved elliptic curves:
  - P-256
  - P-384

**Certificate Management**:
- Self-signed certificates auto-generated at first boot if not provided
- Stored in `/var/lib/boardingpass/tls/` with permissions 0600
- Can be pre-provisioned in bootc image

**Certificate Validation**:
- Clients must accept self-signed certificates (trust on first use)
- Certificate pinning recommended for production deployments
- Future: Support for user-provided CA-signed certificates

---

## Configuration Security

### Path Allow-Lists

Configuration files can only be written to approved directories specified in `/etc/boardingpass/config.yaml`:

```yaml
provisioning:
  allowed_paths:
    - /etc/systemd/system/
    - /etc/NetworkManager/system-connections/
```

**Protections**:
- Path traversal prevention (`..` sequences rejected)
- Absolute path enforcement (relative to `/etc`)
- Symlink resolution disabled
- Validation before any write operations

**Blocked Paths** (examples):
- `/etc/passwd` - User database
- `/etc/shadow` - Password hashes
- `/etc/sudoers` - Sudo configuration
- `/etc/ssh/` - SSH configuration
- Any path not in allow-list

### Atomic Operations

Configuration provisioning uses atomic operations to prevent partial state:

1. **Write to temp directory**: `/var/lib/boardingpass/staging/apply-*`
2. **Validate all files**: Check paths, permissions, sizes
3. **Atomic rename**: Move files to target paths simultaneously
4. **On failure**: Rollback all changes, clean up temp directory

**Benefits**:
- No partial configurations (all-or-nothing)
- No race conditions
- Automatic recovery from failures
- Minimal attack window

### File Permissions

- Configuration files respect requested permissions (mode field)
- Permissions validated before application (must be ≤ 0777)
- No setuid/setgid/sticky bits allowed in mode field

---

## Command Execution Security

### Command Allow-Lists

Only pre-configured commands can be executed, specified in `/etc/boardingpass/config.yaml`:

```yaml
commands:
  - id: "reboot"
    path: "/usr/bin/systemctl"
    args: ["reboot"]
```

**Protections**:
- No arbitrary command execution
- Arguments are fixed (cannot be modified by API calls)
- Commands executed via sudo with restricted permissions
- No shell interpretation (direct execution via exec)

**Sudoers Configuration**:
```
boardingpass ALL=(ALL) NOPASSWD: /usr/bin/systemctl reboot
boardingpass ALL=(ALL) NOPASSWD: /usr/bin/systemctl restart NetworkManager
```

**Security Validation**:
- Sudoers file permissions: 0440 (read-only)
- Sudoers validation on startup
- Command paths must be absolute
- No wildcard expansion in arguments

---

## Logging Security

### Secret Redaction

All logs automatically redact sensitive data:

**Redacted Fields**:
- `password`, `token`, `secret`, `key`
- `proof`, `verifier`, `salt`, `session`
- `content`, `payload`, `authorization`
- SRP ephemeral values (A, B, M1, M2)
- Configuration file content (Base64 payloads)

**Redaction Format**:
```json
{
  "timestamp": "2025-12-09T10:00:00Z",
  "level": "INFO",
  "message": "SRP verification successful",
  "fields": {
    "username": "boardingpass",
    "M1": "[REDACTED]",
    "session_token": "[REDACTED]"
  }
}
```

### Audit Trail

All security-relevant events are logged:

- Authentication attempts (success and failure)
- Session token creation and expiration
- Configuration provisioning (success and failure)
- Command execution (including exit codes)
- Rate limit triggers
- Sentinel file creation
- Service shutdown

**Log Retention**:
- Logs written to stdout/stderr (captured by systemd)
- Retention controlled by systemd journald configuration
- Recommended: 30 days minimum for audit purposes

---

## Lifecycle Security

### Sentinel File

The sentinel file (`/etc/boardingpass/issued`) prevents re-provisioning:

**Properties**:
- Empty file (0 bytes)
- Created after successful configuration provisioning
- Checked at service startup via systemd condition:
  ```
  ConditionPathExists=!/etc/boardingpass/issued
  ```
- Prevents service from starting if file exists

**Security Implications**:
- Once provisioned, device cannot be re-provisioned
- Attacker cannot restart BoardingPass after provisioning
- Manual intervention required to re-provision (testing only)

### Inactivity Timeout

Service terminates after 10 minutes of inactivity (configurable):

**Benefits**:
- Limits attack window
- Prevents orphaned processes
- Ensures ephemeral operation

**Activity Events**:
- API requests (any endpoint)
- Authentication attempts
- Session token validation

### Graceful Shutdown

Service performs graceful shutdown on:

1. Successful configuration provisioning
2. Inactivity timeout
3. Signal (SIGTERM, SIGINT)

**Shutdown Process**:
1. Stop accepting new requests
2. Complete in-flight requests
3. Invalidate all session tokens
4. Clear sensitive data from memory
5. Exit process

---

## Best Practices

### Deployment

1. **Limit Network Access**: Only allow bootstrap operators to connect (firewall rules)
2. **Monitor Logs**: Set up log aggregation and alerting for suspicious activity
3. **Use Strong Passwords**: Combine multiple hardware identifiers for higher entropy
4. **Restrict Paths**: Only allow necessary directories in `allowed_paths`
5. **Minimal Command Set**: Only include essential commands in `commands` allow-list
6. **Production Logging**: Use JSON format for structured parsing

### Bootc Image Security

1. **Unique Salt**: Generate unique salt for each bootc image build
2. **Pre-provision Certificates**: Include TLS certificates in image to avoid runtime generation
3. **Immutable Configuration**: Bake service configuration into image
4. **Remove Verifier**: Delete `/etc/boardingpass/verifier` after provisioning (optional)

### Post-Provisioning

1. **Verify Sentinel File**: Check that `/etc/boardingpass/issued` exists
2. **Remove Sensitive Files**: Delete verifier config if no longer needed
3. **Review Logs**: Check for any suspicious activity during bootstrap
4. **Network Isolation**: Remove bootstrap network access if applicable

---

## Security Validations

### Automated Checks

BoardingPass includes security validations:

- **golangci-lint with gosec**: Static security analysis
- **govulncheck**: Vulnerability scanning for dependencies
- **Constant-time comparisons**: All cryptographic proofs
- **Input validation**: All API requests
- **Path validation**: All file operations
- **Command validation**: All sudo operations

### Manual Audits

Recommended security audits:

1. **Code Review**: Review all changes before deployment
2. **Penetration Testing**: Test authentication, authorization, and configuration provisioning
3. **Dependency Audit**: Review all dependencies for security issues
4. **Configuration Review**: Verify production configurations follow best practices

---

## Incident Response

### Suspicious Activity

If suspicious activity is detected:

1. **Review Logs**: Check journalctl for details
2. **Identify Source**: Determine client IP and request pattern
3. **Block if Necessary**: Add firewall rules to block attacker
4. **Investigate Root Cause**: Determine how attacker gained access
5. **Re-provision if Compromised**: Factory reset and re-deploy

### Compromise Scenarios

**Scenario 1: Brute-Force Attack**
- Rate limiting automatically engages
- Monitor logs for repeated failures from same IP
- Add firewall rule to block attacker IP

**Scenario 2: Configuration Tampering**
- Atomic operations prevent partial state
- Rollback automatically occurs on failure
- Review logs to identify tampered files

**Scenario 3: Command Execution Exploit**
- Allow-list prevents arbitrary commands
- Sudoers configuration limits permissions
- Review logs for unexpected command executions

**Scenario 4: Session Token Theft**
- Tokens expire after 30 minutes
- Service restart invalidates all tokens
- TLS prevents token interception

---

## Compliance

### FIPS 140-3

BoardingPass is designed for FIPS 140-3 compliance:

- **Cryptography**: Go stdlib crypto/* only (FIPS-validated when built with `GOEXPERIMENT=boringcrypto`)
- **Algorithms**: SHA-256, AES-GCM, P-256/P-384 elliptic curves
- **No Third-Party Crypto**: All cryptographic operations use Go stdlib

### Security Standards

BoardingPass follows industry security standards:

- **OWASP Top 10**: Mitigations for all top 10 vulnerabilities
- **CWE/SANS Top 25**: Mitigations for most dangerous software weaknesses
- **NIST Cybersecurity Framework**: Aligns with identify, protect, detect, respond, recover functions

---

## Vulnerability Disclosure

If you discover a security vulnerability in BoardingPass:

1. **Do not disclose publicly** until a patch is available
2. **Report to**: security@boardingpass.dev (or GitHub private security advisory)
3. **Include**: Detailed description, steps to reproduce, impact assessment
4. **Expect**: Response within 48 hours, fix within 30 days

---

## References

- **RFC 5054**: SRP-6a Protocol Specification
- **RFC 8446**: TLS 1.3 Specification
- **FIPS 140-3**: Federal Information Processing Standard for Cryptographic Modules
- **OWASP**: Open Web Application Security Project
- **CWE**: Common Weakness Enumeration

---

**Document Status**: Complete
**Last Updated**: 2025-12-09
