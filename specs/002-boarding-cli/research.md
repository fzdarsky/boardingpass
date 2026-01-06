# Research: Boarding CLI Tool

**Feature**: 002-boarding-cli
**Date**: 2025-12-10
**Purpose**: Document technical research and design decisions for the Boarding CLI implementation

## 1. SRP-6a Client Implementation

###Decision

Implement a pure Go stdlib client-side SRP-6a protocol handler that mirrors the existing server-side implementation in `internal/auth/srp.go`, reusing the same RFC 5054 group parameters (N, g, k) to ensure compatibility.

### Rationale

- **Compatibility**: Must interoperate with existing BoardingPass SRP server implementation
- **FIPS 140-3 Compliance**: Can only use Go stdlib crypto package (no third-party crypto libraries per constitution)
- **Client-side differences**: Client computes ephemeral keypair (a, A) and proofs (M1), server computes (b, B) and verifies
- **Security**: SRP-6a provides mutual authentication without transmitting password over network

### Implementation Approach

**Two-Phase Flow**:
1. **Init Phase** (`POST /auth/srp/init`):
   - Client generates ephemeral private key `a` (random 256-bit value)
   - Client computes ephemeral public key `A = g^a mod N`
   - Client sends username + A to server
   - Server responds with salt + server's ephemeral public key B

2. **Verify Phase** (`POST /auth/srp/verify`):
   - Client derives private key `x = H(salt | H(username | ":" | password))`
   - Client computes shared secret `S = (B - k*g^x)^(a + u*x) mod N` where `u = H(A | B)`
   - Client computes session key `K = H(S)`
   - Client computes proof `M1 = H(H(N) XOR H(g) | H(username) | salt | A | B | K)`
   - Client sends M1 to server
   - Server verifies M1 and responds with proof M2 + session token
   - Client verifies `M2 = H(A | M1 | K)`

**Key Functions** (in `internal/cli/client/srp.go`):
```go
GenerateEphemeralKeypair() (a, A *big.Int)
ComputeSharedSecret(a, B, x *big.Int) (S, K *big.Int)
ComputeClientProof(K, A, B, salt []byte, username string) []byte  // M1
VerifyServerProof(M2, A, M1, K []byte) bool
DerivePrivateKey(salt []byte, username, password string) *big.Int  // x
```

### Alternatives Considered

1. **Use third-party SRP library** (e.g., `github.com/opencoff/go-srp`)
   - **Rejected**: Violates minimal dependencies principle and FIPS 140-3 requirement (no third-party crypto)

2. **Implement different auth protocol** (e.g., OAuth2, JWT)
   - **Rejected**: Server uses SRP-6a; changing protocol would require server changes and break existing clients

3. **Skip client-side SRP, send password directly**
   - **Rejected**: Major security degradation; password would be transmitted over network even with TLS

## 2. Configuration Precedence Mechanism

### Decision

Implement three-layer configuration with explicit precedence: command-line flags (highest) > environment variables > config file (lowest). Missing required values after all sources are consulted results in clear error message.

### Rationale

- **Flexibility**: Different use cases prefer different config methods (devs use flags, CI uses env vars, sysadmins use config files)
- **Principle of Least Surprise**: Follows standard CLI tool patterns (kubectl, aws-cli, docker, git all use similar precedence)
- **Explicit > Implicit**: Command-line flags should override everything (user's immediate intent)
- **FR-022 Compliance**: Spec explicitly requires this precedence order

### Implementation Approach

**Config Loading Sequence** (in `internal/cli/config/config.go`):
```go
type Config struct {
    Host   string
    Port   int
    CACert string // optional
}

func Load() (*Config, error) {
    cfg := &Config{
        Port: 8443, // default
    }

    // Layer 1: Config file (lowest priority)
    if fileConfig, err := loadConfigFile(); err == nil {
        cfg.merge(fileConfig)
    }

    // Layer 2: Environment variables (medium priority)
    if host := os.Getenv("BOARDING_HOST"); host != "" {
        cfg.Host = host
    }
    if port := os.Getenv("BOARDING_PORT"); port != "" {
        cfg.Port = mustParseInt(port)
    }
    if caCert := os.Getenv("BOARDING_CA_CERT"); caCert != "" {
        cfg.CACert = caCert
    }

    // Layer 3: Command-line flags (highest priority)
    // Applied in command-specific Execute() functions

    // Validation
    if cfg.Host == "" {
        return nil, fmt.Errorf("no host specified (use --host flag, BOARDING_HOST env var, or config.yaml)")
    }

    return cfg, nil
}
```

**Config File Location** (per FR-022):
- Linux: `~/.config/boardingpass/config.yaml`
- macOS: `~/Library/Application Support/boardingpass/config.yaml`
- Windows: `%APPDATA%\boardingpass\config.yaml`

Use `os.UserConfigDir()` for portability.

**Config File Format** (flat YAML per FR-021):
```yaml
host: boardingpass.local
port: 8443
ca_cert: /path/to/ca-bundle.pem
```

### Alternatives Considered

1. **Single source (env vars only or flags only)**
   - **Rejected**: Inflexible; forces all users to use same config method

2. **Different precedence order** (e.g., file > env > flags)
   - **Rejected**: Counter-intuitive; command-line should always win (user's immediate intent)

3. **Merge all sources equally**
   - **Rejected**: Ambiguous when multiple sources define same key; precedence eliminates ambiguity

## 3. TLS Trust-on-First-Use (TOFU) Pattern

### Decision

Implement custom `http.RoundTripper` that intercepts TLS handshake to compute certificate SHA-256 fingerprint, prompt user for acceptance on first connection, and store accepted fingerprints in persistent YAML file for subsequent connections. Support `--ca-cert` flag for custom CA bundles (skips TOFU when cert validates against CA).

### Rationale

- **Self-Signed Certs Common**: BoardingPass often uses self-signed certs (spec assumption line 177)
- **Security vs UX Balance**: Strict TLS validation (reject self-signed) breaks UX; auto-accept is insecure; TOFU provides informed consent
- **SSH-like UX**: Users familiar with SSH "unknown host" prompts
- **FR-026, FR-027 Compliance**: Spec explicitly requires prompt on first connection + remember choice

### Implementation Approach

**Custom Transport** (in `internal/cli/client/transport.go`):
```go
type FingerprintTransport struct {
    base      *http.Transport
    certStore *tls.CertificateStore
}

func (t *FingerprintTransport) RoundTrip(req *http.Request) (*http.Response, error) {
    // Let base transport do TLS handshake
    resp, err := t.base.RoundTrip(req)

    // On TLS error, extract cert and check fingerprint
    if isTLSError(err) {
        cert := extractCert(err)
        fingerprint := computeSHA256Fingerprint(cert)

        // Check if fingerprint is known
        if !t.certStore.IsKnown(req.Host, fingerprint) {
            // Prompt user
            if !promptAcceptCert(req.Host, fingerprint) {
                return nil, fmt.Errorf("certificate rejected by user")
            }
            t.certStore.Add(req.Host, fingerprint)
        }

        // Retry with InsecureSkipVerify
        t.base.TLSClientConfig.InsecureSkipVerify = true
        resp, err = t.base.RoundTrip(req)
    }

    return resp, err
}
```

**Fingerprint Storage** (in `internal/cli/tls/store.go`):
File: `<UserConfigDir>/boardingpass/known_certs.yaml`
```yaml
certificates:
  - host: "192.168.1.100:8443"
    fingerprint: "SHA256:abc123def456..."
    accepted_at: "2025-12-10T12:00:00Z"
  - host: "boardingpass.local:8443"
    fingerprint: "SHA256:789ghi012jkl..."
    accepted_at: "2025-12-10T13:30:00Z"
```

**Custom CA Support** (FR-028):
```go
if caCertPath != "" {
    caCert, _ := os.ReadFile(caCertPath)
    certPool := x509.SystemCertPool()
    certPool.AppendCertsFromPEM(caCert)
    transport.TLSClientConfig.RootCAs = certPool
    // Skip TOFU if cert validates against CA
}
```

### Alternatives Considered

1. **Auto-accept all self-signed certs**
   - **Rejected**: Insecure; exposes to MITM attacks

2. **Require `--insecure` flag for self-signed**
   - **Rejected**: Poor UX; user must remember flag every time

3. **Cert pinning (reject on fingerprint change)**
   - **Rejected**: Too strict; legitimate cert rotation would break

4. **Use OS keychain for cert storage**
   - **Rejected**: Platform-specific complexity; YAML file is simpler and cross-platform

## 4. Session Token Storage Strategy

### Decision

Store session tokens in OS-specific cache directory (`os.UserCacheDir()/boardingpass/`) with filename pattern `session-<hash-of-host-port>.token` and file permissions `0600` (owner read/write only). Tokens persist across CLI invocations but are cleared on system reboot. Support multiple concurrent sessions to different BoardingPass servers via per-server token files.

### Rationale

- **Cross-Platform**: `os.UserCacheDir()` provides OS-appropriate temp storage (Linux: `~/.cache`, macOS: `~/Library/Caches`, Windows: `%LocalAppData%`)
- **Ephemeral**: Cache dirs are cleared on reboot, meeting constitution's ephemeral principle
- **Secure**: 0600 permissions prevent other users from reading tokens (FR-005)
- **Multi-Server**: Hash of host:port in filename allows sessions to multiple services (developer may provision multiple devices)
- **FR-004, FR-006 Compliance**: Spec requires OS temp dir + per-user storage

### Implementation Approach

**Token File Paths**:
```
Linux:   ~/.cache/boardingpass/session-<sha256-of-host-port>.token
macOS:   ~/Library/Caches/boardingpass/session-<sha256-of-host-port>.token
Windows: %LocalAppData%\boardingpass\cache\session-<sha256-of-host-port>.token
```

**Session Store** (in `internal/cli/session/store.go`):
```go
type Store struct {
    dir string  // os.UserCacheDir()/boardingpass/
}

func (s *Store) Save(host, port, token string) error {
    filename := s.tokenFilename(host, port)
    if err := os.MkdirAll(s.dir, 0700); err != nil {
        return err
    }
    if err := os.WriteFile(filename, []byte(token), 0600); err != nil {
        return err
    }
    return nil
}

func (s *Store) Load(host, port string) (string, error) {
    filename := s.tokenFilename(host, port)
    data, err := os.ReadFile(filename)
    if err != nil {
        return "", err
    }
    return string(data), nil
}

func (s *Store) Delete(host, port string) error {
    filename := s.tokenFilename(host, port)
    return os.Remove(filename)
}

func (s *Store) tokenFilename(host, port string) string {
    hash := sha256.Sum256([]byte(host + ":" + port))
    return filepath.Join(s.dir, fmt.Sprintf("session-%x.token", hash[:8]))
}
```

**File Format**: Single-line text file containing just the session token (no JSON/YAML overhead for performance).

### Alternatives Considered

1. **In-memory only (no persistence)**
   - **Rejected**: Every CLI command would require re-authentication; violates SC-007 (transparent session management)

2. **Persistent storage in user config dir**
   - **Rejected**: Config dir is for persistent configuration; session tokens are ephemeral; spec requires temp dir (FR-004)

3. **Single token file for all servers**
   - **Rejected**: Cannot support concurrent sessions to multiple services; limits developer workflow

4. **Environment variable for token**
   - **Rejected**: Environment variables persist in shell history and process listings (security risk)

## 5. Go Stdlib Flag Package vs External CLI Frameworks

### Decision

Use Go stdlib `flag` package for command-line parsing with manual command routing via `switch` statement. Explicitly REJECT external CLI frameworks (cobra, urfave/cli, etc.).

### Rationale

- **Constitution Compliance**: Minimal Dependencies principle requires justification for all external dependencies
- **Existing Pattern**: `cmd/boardingpass/main.go` uses stdlib `flag` package; CLI should follow same pattern
- **Project Policy**: CLAUDE.md states "gopkg.in/yaml.v3 is the only allowed external runtime dependency"
- **Simplicity**: CLI has only 6 commands; framework overhead not justified

### Implementation Approach

**Command Routing** (in `cmd/boarding/main.go`):
```go
func main() {
    if len(os.Args) < 2 {
        printUsage()
        os.Exit(1)
    }

    command := os.Args[1]
    switch command {
    case "pass":
        commands.NewPassCommand().Execute(os.Args[2:])
    case "info":
        commands.NewInfoCommand().Execute(os.Args[2:])
    case "connections":
        commands.NewConnectionsCommand().Execute(os.Args[2:])
    case "load":
        commands.NewLoadCommand().Execute(os.Args[2:])
    case "command":
        commands.NewCommandCommand().Execute(os.Args[2:])
    case "complete":
        commands.NewCompleteCommand().Execute(os.Args[2:])
    default:
        fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
        printUsage()
        os.Exit(1)
    }
}
```

**Per-Command Flags** (example: `internal/cli/commands/pass.go`):
```go
func (c *PassCommand) Execute(args []string) {
    fs := flag.NewFlagSet("pass", flag.ExitOnError)
    username := fs.String("username", "", "Username for authentication")
    password := fs.String("password", "", "Password for authentication")
    host := fs.String("host", "", "BoardingPass service host")
    port := fs.Int("port", 0, "BoardingPass service port")

    fs.Parse(args)

    // Merge flags with config (flags override config)
    cfg := config.Load()
    if *host != "" {
        cfg.Host = *host
    }
    if *port != 0 {
        cfg.Port = *port
    }

    // Execute authentication logic
    // ...
}
```

### Alternatives Considered

1. **Use cobra framework**
   - **Rejected**: Adds ~20 external dependencies; violates minimal dependencies principle; overkill for 6 commands

2. **Use urfave/cli framework**
   - **Rejected**: Still an external dependency; project policy is stdlib only except yaml

3. **Implement custom mini-framework**
   - **Rejected**: NIH syndrome; stdlib `flag` is sufficient and well-tested

## 6. Output Formatting (YAML vs JSON)

### Decision

Default to YAML output for `info` and `connections` commands (FR-009), with `-o json` flag to switch to JSON (FR-010). Implement both formatters using Go stdlib + gopkg.in/yaml.v3 (already present).

### Rationale

- **Human Readability**: YAML is more readable for developers doing manual inspection
- **Machine Parsability**: JSON is better for CI scripts and jq processing
- **Spec Requirement**: FR-009 and FR-010 explicitly define this behavior
- **Existing Dependency**: yaml.v3 already in go.mod for config parsing; no new dependency

### Implementation Approach

**Formatter** (in `internal/cli/output/formatter.go`):
```go
func FormatYAML(v interface{}) ([]byte, error) {
    return yaml.Marshal(v)
}

func FormatJSON(v interface{}) ([]byte, error) {
    return json.MarshalIndent(v, "", "  ")
}
```

**Usage in Commands**:
```go
var data protocol.InfoResponse
// ... fetch data from API ...

var output []byte
if outputFormat == "json" {
    output, _ = output.FormatJSON(data)
} else {
    output, _ = output.FormatYAML(data)
}
fmt.Println(string(output))
```

### Alternatives Considered

1. **JSON only (no YAML)**
   - **Rejected**: Spec requires YAML as default for better human readability

2. **Table output (like kubectl)**
   - **Rejected**: Spec only mentions YAML and JSON; table format would require significant parsing logic

3. **Use text/template for custom formats**
   - **Rejected**: YAGNI; spec doesn't require custom formats

## Summary of Key Decisions

| Area | Decision | Primary Rationale |
|------|----------|-------------------|
| SRP Client | Stdlib implementation mirroring server | FIPS 140-3 compliance, no third-party crypto |
| Config Precedence | Flags > Env > File | Standard CLI pattern, explicit user intent wins |
| TLS TOFU | Custom RoundTripper with fingerprint prompts | Balance security and UX for self-signed certs |
| Session Storage | OS cache dir with 0600 permissions | Ephemeral, secure, multi-server support |
| CLI Framework | Stdlib flag package, manual routing | Minimal dependencies, follows existing pattern |
| Output Format | YAML default, JSON via flag | Human-readable default, machine-parsable option |

**All decisions align with constitution principles: minimal dependencies, FIPS compliance, ephemeral operation, fail-safe design.**
