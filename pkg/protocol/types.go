package protocol

// SystemInfo represents hardware and software characteristics of the device.
// Derived from system inspection (/sys, /proc, DMI, TPM).
type SystemInfo struct {
	TPM   TPMInfo   `json:"tpm"`
	Board BoardInfo `json:"board"`
	CPU   CPUInfo   `json:"cpu"`
	OS    OSInfo    `json:"os"`
}

// TPMInfo represents TPM (Trusted Platform Module) information.
type TPMInfo struct {
	Present      bool    `json:"present"`
	Manufacturer *string `json:"manufacturer"`
	Model        *string `json:"model"`
	Version      *string `json:"version"`
}

// BoardInfo represents motherboard/baseboard information from DMI.
type BoardInfo struct {
	Manufacturer string `json:"manufacturer"`
	Model        string `json:"model"`
	Serial       string `json:"serial"`
}

// CPUInfo represents CPU architecture information.
type CPUInfo struct {
	Architecture string `json:"architecture"`
}

// OSInfo represents operating system information.
type OSInfo struct {
	Distribution string `json:"distribution"`
	Version      string `json:"version"`
	FIPSEnabled  bool   `json:"fips_enabled"`
}

// NetworkConfig represents current network interface state.
// Real-time snapshot queried from the system.
type NetworkConfig struct {
	Interfaces []NetworkInterface `json:"interfaces"`
}

// NetworkInterface represents a single network interface.
type NetworkInterface struct {
	Name        string      `json:"name"`
	MACAddress  string      `json:"mac_address"`
	LinkState   string      `json:"link_state"`
	IPAddresses []IPAddress `json:"ip_addresses"`
}

// IPAddress represents an IP address assignment.
type IPAddress struct {
	IP     string `json:"ip"`
	Prefix int    `json:"prefix"`
	Family string `json:"family"`
}

// ConfigBundle represents a collection of files to be atomically written.
type ConfigBundle struct {
	Files []ConfigFile `json:"files"`
}

// ConfigFile represents a single file to be written.
type ConfigFile struct {
	Path    string `json:"path"`
	Content string `json:"content"` // Base64-encoded
	Mode    int    `json:"mode"`    // Unix file permissions
}

// CommandRequest represents a request to execute an allow-listed command.
type CommandRequest struct {
	ID string `json:"id"`
}

// CommandResponse represents the result of command execution.
type CommandResponse struct {
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
}

// SRPInitRequest represents the initial SRP-6a authentication request.
type SRPInitRequest struct {
	Username string `json:"username"`
	A        string `json:"A"` // Base64-encoded ephemeral public key
}

// SRPInitResponse represents the response to SRP init request.
type SRPInitResponse struct {
	Salt string `json:"salt"` // Base64-encoded salt
	B    string `json:"b"`    // Base64-encoded server ephemeral public key
}

// SRPVerifyRequest represents the SRP verification request.
type SRPVerifyRequest struct {
	M1 string `json:"M1"` // Base64-encoded client proof
}

// SRPVerifyResponse represents the response to SRP verify request.
type SRPVerifyResponse struct {
	M2           string `json:"M2"`            // Base64-encoded server proof
	SessionToken string `json:"session_token"` // HMAC-signed session token
}

// CompleteResponse represents the response to POST /complete.
type CompleteResponse struct {
	Status       string  `json:"status"`
	SentinelFile string  `json:"sentinel_file"`
	Message      *string `json:"message,omitempty"`
}
