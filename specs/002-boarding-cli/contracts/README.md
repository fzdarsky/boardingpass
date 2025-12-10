# API Contracts: Boarding CLI Tool

**Feature**: 002-boarding-cli
**Date**: 2025-12-10

## Overview

The Boarding CLI tool is a **pure client** that consumes the existing BoardingPass REST API. It does NOT define new API contracts or endpoints.

## API Consumer Role

The CLI interacts with the following existing BoardingPass API endpoints:

### Authentication Endpoints

**POST /auth/srp/init**
- **Purpose**: Initiate SRP-6a authentication (Phase 1)
- **Request**: `{ "username": "admin", "A": "<client-ephemeral-public-key>" }`
- **Response**: `{ "salt": "<salt>", "B": "<server-ephemeral-public-key>" }`
- **Used by**: `boarding pass` command

**POST /auth/srp/verify**
- **Purpose**: Complete SRP-6a authentication (Phase 2)
- **Request**: `{ "M1": "<client-proof>" }`
- **Response**: `{ "M2": "<server-proof>", "session_token": "<jwt-or-opaque-token>" }`
- **Used by**: `boarding pass` command

### Data Query Endpoints

**GET /info**
- **Purpose**: Query system inventory (CPU, board, TPM, OS, FIPS status)
- **Authentication**: Requires `Authorization: Bearer <session-token>` header
- **Response**: JSON object with system information
- **Used by**: `boarding info` command

**GET /network**
- **Purpose**: Query network interface configuration
- **Authentication**: Requires `Authorization: Bearer <session-token>` header
- **Response**: JSON array of network interfaces
- **Used by**: `boarding connections` command

### Provisioning Endpoints

**POST /configure**
- **Purpose**: Upload configuration bundle (directory of files)
- **Authentication**: Requires `Authorization: Bearer <session-token>` header
- **Request**: Multipart form data with files
- **Response**: Status confirmation
- **Used by**: `boarding load` command
- **Note**: As of spec date, this endpoint is marked TODO in `internal/api/handlers/` - implementation pending

**POST /command**
- **Purpose**: Execute allow-listed command on device
- **Authentication**: Requires `Authorization: Bearer <session-token>` header
- **Request**: `{ "command": "systemctl restart networking" }`
- **Response**: `{ "stdout": "...", "stderr": "...", "exit_code": 0 }`
- **Used by**: `boarding command` command
- **Note**: As of spec date, this endpoint is marked TODO in `internal/api/handlers/` - implementation pending

**POST /complete**
- **Purpose**: Signal provisioning complete, trigger sentinel file creation
- **Authentication**: Requires `Authorization: Bearer <session-token>` header
- **Response**: Status confirmation
- **Used by**: `boarding complete` command

## Contract Definitions

The formal API contract definitions are maintained in the BoardingPass service repository:

**OpenAPI Specification**: `specs/001-boardingpass-api/contracts/openapi.yaml`

The CLI implementation references `pkg/protocol/types.go` for request/response struct definitions, ensuring type safety and consistency with the server implementation.

## Client-Side Protocol Implementation

While the CLI does NOT define new API endpoints, it DOES implement the **client-side** of the SRP-6a authentication protocol:

**SRP-6a Client Protocol** (RFC 5054):
```
Phase 1 (Init):
  Client: Generate random ephemeral private key a
  Client: Compute ephemeral public key A = g^a mod N
  Client → Server: username, A
  Server → Client: salt, B (server ephemeral public key)

Phase 2 (Verify):
  Client: Derive x = H(salt | H(username | ":" | password))
  Client: Compute u = H(A | B)
  Client: Compute S = (B - k*g^x)^(a + u*x) mod N
  Client: Compute K = H(S)  // session key
  Client: Compute M1 = H(H(N) XOR H(g) | H(username) | salt | A | B | K)
  Client → Server: M1 (client proof)
  Server → Client: M2 (server proof), session_token
  Client: Verify M2 = H(A | M1 | K)
```

**Implementation**: `internal/cli/client/srp.go`

**Parameters** (must match server-side `internal/auth/srp.go`):
- N (RFC 5054 3072-bit group)
- g (generator = 5)
- k (multiplier = H(N | g))
- Hash function (SHA-256)

## Compatibility Requirements

The CLI MUST maintain compatibility with the BoardingPass service API:

1. **SRP Parameters**: Must use identical N, g, k values as server
2. **HTTP Methods**: Must use correct HTTP verbs for each endpoint
3. **Request/Response Format**: Must match JSON schema defined in server
4. **Authentication Header**: Must include `Authorization: Bearer <token>` for authenticated endpoints
5. **TLS Version**: Must support TLS 1.3+ (server requirement)

## Version Compatibility Matrix

| CLI Version | Compatible BoardingPass Service Version |
|-------------|------------------------------------------|
| 1.0.x       | 1.0.x (initial release)                  |

**Note**: CLI and service SHOULD be deployed at matching major.minor versions. Patch versions are compatible within the same major.minor series.

## Error Handling

The CLI handles standard HTTP error responses from the API:

| Status Code | Meaning | CLI Action |
|-------------|---------|------------|
| 200 OK | Success | Display formatted response |
| 401 Unauthorized | Invalid/expired session token | Delete token, prompt to re-authenticate |
| 403 Forbidden | Command not in allow-list, path not allowed | Display error message |
| 404 Not Found | Endpoint not found | Display error (possible version mismatch) |
| 500 Internal Server Error | Server error | Display error, suggest contacting admin |
| 503 Service Unavailable | Service down/overloaded | Display error, suggest retry |

Error responses follow the format:
```json
{
  "error": "error_code_constant",
  "message": "Human-readable error description"
}
```

Defined in `pkg/protocol/errors.go`.

## API Discovery

The CLI does NOT implement API discovery or version negotiation. It assumes:
- API endpoint paths are fixed (per OpenAPI spec)
- API is available at `https://<host>:<port>/`
- No `/api/v1/` style versioning prefix

Future versions MAY add API versioning support if the server implements versioned endpoints.

## Testing

API contract compliance is verified through integration tests:

**Location**: `tests/cli-integration/`

**Approach**:
- Mock server using `httptest` package
- Simulate all API endpoints with correct request/response formats
- Verify CLI sends correct requests and handles responses properly

**E2E Testing**:
- Tests against real BoardingPass service in `tests/cli-e2e/`
- Ensures actual API compatibility (not just mock compatibility)

## References

- BoardingPass API Specification: `specs/001-boardingpass-api/spec.md`
- OpenAPI Definition: `specs/001-boardingpass-api/contracts/openapi.yaml`
- Protocol Types: `pkg/protocol/types.go`
- SRP-6a RFC: RFC 5054 (https://tools.ietf.org/html/rfc5054)
