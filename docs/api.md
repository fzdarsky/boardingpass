# BoardingPass API Documentation

**Version**: 0.1.0
**Last Updated**: 2025-12-09

## Overview

BoardingPass exposes a RESTful API over HTTPS (TLS 1.3 required) for secure device provisioning. All endpoints except authentication return JSON responses.

**Base URL**: `https://{device_ip}:8443`

---

## Authentication

BoardingPass uses SRP-6a (Secure Remote Password) for mutual authentication without PKI. The authentication flow has two steps:

1. **Initialize**: POST `/auth/srp/init` - Exchange username and ephemeral public values
2. **Verify**: POST `/auth/srp/verify` - Verify proofs and obtain session token

After authentication, include the session token in all subsequent requests:

```
Authorization: Bearer {session_token}
```

Session tokens expire after 30 minutes (configurable).

---

## Endpoints

### Authentication

#### POST /auth/srp/init

Initialize SRP-6a handshake.

**Request**:
```json
{
  "username": "boardingpass",
  "A": "dGhpc2lzYW5leGFtcGxlZXBoZW1lcmFscHVibGljdmFsdWVB..."
}
```

**Response**:
```json
{
  "salt": "c29tZXJhbmRvbXNhbHR2YWx1ZQ==",
  "B": "dGhpc2lzYW5leGFtcGxlZXBoZW1lcmFscHVibGljdmFsdWVC..."
}
```

**Status Codes**:
- `200 OK`: Handshake initialized
- `400 Bad Request`: Invalid request format
- `500 Internal Server Error`: Server error

---

#### POST /auth/srp/verify

Verify SRP-6a proof and obtain session token.

**Request**:
```json
{
  "M1": "dGhpc2lzYW5leGFtcGxlY2xpZW50cHJvb2Y="
}
```

**Response**:
```json
{
  "M2": "dGhpc2lzYW5leGFtcGxlc2VydmVycHJvb2Y=",
  "session_token": "dGhpc2lzYXRva2VuaWQ.c2lnbmF0dXJlaGVyZQ"
}
```

**Status Codes**:
- `200 OK`: Verification successful, session token issued
- `400 Bad Request`: Invalid request format
- `401 Unauthorized`: Invalid proof (authentication failed)
- `429 Too Many Requests`: Rate limit exceeded (progressive delays: 1s, 2s, 5s, 60s lockout)
- `500 Internal Server Error`: Server error

---

### Device Information

#### GET /info

Query device system information.

**Authentication**: Required

**Response**:
```json
{
  "tpm": {
    "present": true,
    "manufacturer": "STMicroelectronics",
    "model": "ST33HTPH2E32",
    "version": "2.0"
  },
  "board": {
    "manufacturer": "Raspberry Pi Foundation",
    "model": "Raspberry Pi 4 Model B",
    "serial": "10000000abcdef01"
  },
  "cpu": {
    "architecture": "aarch64"
  },
  "os": {
    "distribution": "Red Hat Enterprise Linux",
    "version": "9.3"
  },
  "fips_mode": true
}
```

**Status Codes**:
- `200 OK`: System information retrieved
- `401 Unauthorized`: Missing or invalid session token
- `500 Internal Server Error`: Server error

---

#### GET /network

Query network interface state.

**Authentication**: Required

**Response**:
```json
{
  "interfaces": [
    {
      "name": "eth0",
      "mac": "dc:a6:32:12:34:56",
      "link_state": "up",
      "addresses": [
        {
          "ip": "192.168.1.100",
          "prefix": 24,
          "family": "ipv4"
        },
        {
          "ip": "fe80::dea6:32ff:fe12:3456",
          "prefix": 64,
          "family": "ipv6"
        }
      ]
    }
  ]
}
```

**Status Codes**:
- `200 OK`: Network configuration retrieved
- `401 Unauthorized`: Missing or invalid session token
- `500 Internal Server Error`: Server error

---

### Configuration Provisioning

#### POST /configure

Provision configuration bundle atomically.

**Authentication**: Required

**Request**:
```json
{
  "files": [
    {
      "path": "systemd/network/10-eth0.network",
      "content": "W01hdGNoXQpOYW1lPWV0aDAKCltOZXR3b3JrXQpBZGRyZXNzPTE5Mi4xNjguMS4xMDAvMjQKR2F0ZXdheT0xOTIuMTY4LjEuMQpETlM9OC44LjguOAo=",
      "mode": 420
    }
  ]
}
```

**Notes**:
- `path`: Relative to `/etc` (e.g., `systemd/network/10-eth0.network` → `/etc/systemd/network/10-eth0.network`)
- `content`: Base64-encoded file content
- `mode`: Unix file permissions as decimal (e.g., 420 = 0644 octal)
- Paths must be in the `allowed_paths` allow-list configured in `/etc/boardingpass/config.yaml`
- Maximum bundle size: 10MB (total decoded content)
- Maximum file count: 100 files

**Response**:
```json
{
  "status": "success",
  "message": "Configuration bundle applied successfully. Service will terminate."
}
```

**Status Codes**:
- `200 OK`: Configuration applied successfully
- `400 Bad Request`: Invalid request format, path not allowed, bundle too large, or too many files
- `401 Unauthorized`: Missing or invalid session token
- `500 Internal Server Error`: Configuration application failed (rollback performed)

---

### Command Execution

#### POST /command

Execute allow-listed command.

**Authentication**: Required

**Request**:
```json
{
  "id": "restart-networkmanager"
}
```

**Notes**:
- `id`: Command identifier from the allow-list in `/etc/boardingpass/config.yaml`
- No arbitrary commands permitted
- Command arguments are fixed in configuration

**Response**:
```json
{
  "exit_code": 0,
  "stdout": "",
  "stderr": ""
}
```

**Status Codes**:
- `200 OK`: Command executed (check `exit_code` for success/failure)
- `400 Bad Request`: Invalid request format
- `401 Unauthorized`: Missing or invalid session token
- `403 Forbidden`: Command not in allow-list
- `500 Internal Server Error`: Server error

---

### Lifecycle Management

#### POST /complete

Signal provisioning completion.

**Authentication**: Required

**Request**: Empty body

**Response**:
```json
{
  "status": "success",
  "message": "Provisioning complete. Service terminating."
}
```

**Notes**:
- Creates sentinel file (`/etc/boardingpass/issued`)
- Initiates graceful shutdown
- Service will not start again (sentinel file prevents it)

**Status Codes**:
- `200 OK`: Provisioning completed, service shutting down
- `401 Unauthorized`: Missing or invalid session token
- `500 Internal Server Error`: Server error

---

## Error Responses

All error responses follow this format:

```json
{
  "error": "error_code",
  "message": "Human-readable error message"
}
```

### Common Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `invalid_request` | 400 | Malformed JSON or missing required fields |
| `authentication_failed` | 401 | SRP proof verification failed |
| `unauthorized` | 401 | Missing or invalid session token |
| `session_expired` | 401 | Session token has expired |
| `rate_limit_exceeded` | 429 | Too many failed authentication attempts |
| `path_not_allowed` | 400 | File path not in allow-list |
| `command_forbidden` | 403 | Command not in allow-list |
| `bundle_too_large` | 400 | Configuration bundle exceeds 10MB limit |
| `too_many_files` | 400 | Configuration bundle exceeds 100 files |
| `provisioning_failed` | 500 | Failed to apply configuration bundle |
| `internal_error` | 500 | Unexpected server error |

---

## Rate Limiting

Authentication endpoints (`/auth/srp/init` and `/auth/srp/verify`) implement progressive rate limiting per client IP:

1. **1st failure**: 1-second delay
2. **2nd failure**: 2-second delay
3. **3rd failure**: 5-second delay
4. **4th+ failures**: 60-second lockout

Failed authentication responses include a `Retry-After` header with the delay in seconds.

Rate limit state resets after successful authentication.

---

## Examples

### Complete Authentication Flow

```bash
# 1. Initialize SRP handshake
curl -k -X POST https://192.168.1.100:8443/auth/srp/init \
  -H "Content-Type: application/json" \
  -d '{"username":"boardingpass","A":"..."}'

# Response: {"salt":"...","B":"..."}

# 2. Verify proof and obtain session token
curl -k -X POST https://192.168.1.100:8443/auth/srp/verify \
  -H "Content-Type: application/json" \
  -d '{"M1":"..."}'

# Response: {"M2":"...","session_token":"xyz.abc"}

# 3. Use session token for authenticated requests
export SESSION_TOKEN="xyz.abc"
```

### Query Device Information

```bash
curl -k -H "Authorization: Bearer $SESSION_TOKEN" \
  https://192.168.1.100:8443/info
```

### Provision Configuration

```bash
# Encode file content as Base64
CONTENT=$(cat my-service.service | base64 -w0)

# Create configuration bundle
cat <<EOF > bundle.json
{
  "files": [
    {
      "path": "systemd/system/my-service.service",
      "content": "$CONTENT",
      "mode": 420
    }
  ]
}
EOF

# Apply configuration
curl -k -X POST \
  -H "Authorization: Bearer $SESSION_TOKEN" \
  -H "Content-Type: application/json" \
  -d @bundle.json \
  https://192.168.1.100:8443/configure
```

### Execute Command

```bash
curl -k -X POST \
  -H "Authorization: Bearer $SESSION_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"id":"restart-networkmanager"}' \
  https://192.168.1.100:8443/command
```

### Complete Provisioning

```bash
curl -k -X POST \
  -H "Authorization: Bearer $SESSION_TOKEN" \
  https://192.168.1.100:8443/complete
```

---

## Security Considerations

### TLS Configuration

- **TLS 1.3 required**: TLS 1.2 and below are rejected
- **Cipher suites**: AES-128-GCM-SHA256, AES-256-GCM-SHA384 (FIPS-approved)
- **Curves**: P-256, P-384 (FIPS-approved)
- **Certificates**: Self-signed certificates auto-generated if not provided

### Authentication

- **SRP-6a**: Provides mutual authentication and perfect forward secrecy
- **Device-unique passwords**: Generated from hardware identifiers (serial number, TPM, MAC address)
- **Session tokens**: HMAC-signed, 30-minute TTL, in-memory only (not persisted)
- **Rate limiting**: Progressive delays prevent brute-force attacks

### Authorization

- **Path allow-lists**: Configuration files can only be written to approved directories
- **Command allow-lists**: Only pre-configured commands can be executed
- **No arbitrary execution**: Command arguments are fixed in configuration

### Logging

- **Secret redaction**: All sensitive data (passwords, tokens, proofs, configuration content) is automatically redacted from logs
- **Structured logging**: JSON format for machine parsing
- **Audit trail**: All authentication attempts, configuration changes, and command executions are logged

---

## References

- **OpenAPI Specification**: [../specs/001-boardingpass-api/contracts/openapi.yaml](../specs/001-boardingpass-api/contracts/openapi.yaml)
- **Deployment Guide**: [deployment.md](deployment.md)
- **Development Guide**: [development.md](development.md)
- **Security Guide**: [security.md](security.md)

---

**Document Status**: Complete
**Last Updated**: 2025-12-09
