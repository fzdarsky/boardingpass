# BoardingPass API Examples

This directory contains example requests and responses for the BoardingPass API. All examples use realistic data structures and Base64-encoded content where appropriate.

## Authentication Examples

### auth-srp-init.json

SRP-6a handshake initialization (Step 1).

**Request**: Client sends username and ephemeral public value `A`
**Response**: Server returns salt and ephemeral public value `B`

**Note**: SRP values (`A`, `B`, salt) are Base64-encoded large integers (2048-bit).

### auth-srp-verify.json

SRP-6a handshake verification (Step 2).

**Request**: Client sends proof `M1`
**Response**: Server returns proof `M2` and session token

**Note**: Session token format is `<token_id>.<signature>` and should be included in the `Authorization: Bearer <token>` header for all subsequent requests.

## Device Information Examples

### info-response.json

System information response from `GET /info` endpoint.

**Contains**:
- TPM information (manufacturer, model, version)
- Board information (manufacturer, model, serial number)
- CPU architecture
- OS distribution and version
- FIPS mode status

**Example Device**: Raspberry Pi 4 Model B with TPM 2.0, running RHEL 9.3 in FIPS mode

### network-response.json

Network configuration response from `GET /network` endpoint.

**Contains**:
- Network interfaces (eth0, wlan0, lo)
- MAC addresses
- Link states (up/down)
- IP addresses (IPv4 and IPv6) with prefix lengths

**Example Configuration**:
- `eth0`: UP, configured with static IP `192.168.1.100/24`
- `wlan0`: DOWN, no IP addresses
- `lo`: UP, loopback interface

## Configuration Provisioning Examples

### configure-request.json

Configuration bundle provisioning request to `POST /configure` endpoint.

**Contains**: 3 files to be atomically written to `/etc`:

1. **systemd/network/10-eth0.network** (mode: 0644)
   ```ini
   [Match]
   Name=eth0

   [Network]
   Address=192.168.1.100/24
   Gateway=192.168.1.1
   DNS=8.8.8.8
   DNS=8.8.4.4
   ```

2. **chrony/chrony.conf** (mode: 0644)
   ```
   server time.cloudflare.com iburst
   server time.google.com iburst

   driftfile /var/lib/chrony/drift
   makestep 1.0 3
   rtcsync
   ```

3. **hostname** (mode: 0644)
   ```
   boardingpass-device-01
   ```

**Note**: All file content is Base64-encoded in the API request.

## Command Execution Examples

### command-request.json

Command execution request to `POST /command` endpoint.

**Request**: Execute `restart-networkmanager` command from allow-list

**Response (Success)**:
- Exit code: 0
- No stdout/stderr (service restarted silently)

**Response (Failure)**:
- Exit code: 1
- stderr contains error message

**Allow-List Commands** (configured in `/etc/boardingpass/config.yaml`):
- `reboot`: `/usr/bin/systemctl reboot --force`
- `restart-networkmanager`: `/usr/bin/systemctl restart NetworkManager`
- `restart-chronyd`: `/usr/bin/systemctl restart chronyd`

## Decoding Base64 Content

To decode Base64 content from examples:

**Bash**:
```bash
echo "W01hdGNoXQpOYW1lPWV0aDAK..." | base64 -d
```

**Python**:
```python
import base64
content = "W01hdGNoXQpOYW1lPWV0aDAK..."
decoded = base64.b64decode(content).decode('utf-8')
print(decoded)
```

**Go**:
```go
import "encoding/base64"

content := "W01hdGNoXQpOYW1lPWV0aDAK..."
decoded, _ := base64.StdEncoding.DecodeString(content)
fmt.Println(string(decoded))
```

## Testing with curl

### Authentication Flow

```bash
# Step 1: SRP Init (TODO: requires SRP client library to generate 'A')
curl -k -X POST https://192.168.1.100:8443/auth/srp/init \
  -H "Content-Type: application/json" \
  -d '{"username":"boardingpass","A":"<generated_value>"}'

# Step 2: SRP Verify (TODO: requires SRP client library to compute M1)
curl -k -X POST https://192.168.1.100:8443/auth/srp/verify \
  -H "Content-Type: application/json" \
  -d '{"M1":"<computed_value>"}'
```

### Authenticated Requests

```bash
# Save session token
TOKEN="dGhpc2lzYXRva2VuaWQ.c2lnbmF0dXJlaGVyZQ"

# Get system info
curl -k -X GET https://192.168.1.100:8443/info \
  -H "Authorization: Bearer $TOKEN"

# Get network config
curl -k -X GET https://192.168.1.100:8443/network \
  -H "Authorization: Bearer $TOKEN"

# Provision configuration bundle
curl -k -X POST https://192.168.1.100:8443/configure \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d @configure-request.json

# Execute command
curl -k -X POST https://192.168.1.100:8443/command \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"id":"restart-networkmanager"}'
```

**Note**: `-k` flag disables TLS certificate verification (for self-signed certificates in development). Remove in production with valid certificates.

## Error Responses

All error responses follow this schema:

```json
{
  "error": "machine_readable_error_code",
  "message": "Human-readable error message",
  "details": {
    "optional": "context"
  }
}
```

**Common Error Codes**:
- `invalid_request`: Malformed JSON or missing required fields (400)
- `unauthorized`: Missing or invalid session token (401)
- `authentication_failed`: SRP verification failed (401)
- `command_forbidden`: Command not in allow-list (403)
- `provisioning_failed`: Configuration bundle application failed (500)
- `internal_server_error`: Unexpected server error (500)
