package protocol_test

import (
	"encoding/json"
	"testing"

	"github.com/fzdarsky/boardingpass/pkg/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSystemInfo_JSON(t *testing.T) {
	tests := []struct {
		name     string
		input    protocol.SystemInfo
		expected string
	}{
		{
			name: "with TPM present",
			input: protocol.SystemInfo{
				TPM: protocol.TPMInfo{
					Present:      true,
					Manufacturer: stringPtr("STMicroelectronics"),
					Model:        stringPtr("ST33HTPH2E32"),
					Version:      stringPtr("2.0"),
				},
				Board: protocol.BoardInfo{
					Manufacturer: "Raspberry Pi Foundation",
					Model:        "Raspberry Pi 4 Model B",
					Serial:       "10000000abcdef01",
				},
				CPU: protocol.CPUInfo{
					Architecture: "aarch64",
				},
				OS: protocol.OSInfo{
					Distribution: "Red Hat Enterprise Linux",
					Version:      "9.3",
					FIPSEnabled:  true,
				},
			},
			expected: `{"tpm":{"present":true,"manufacturer":"STMicroelectronics","model":"ST33HTPH2E32","version":"2.0"},"board":{"manufacturer":"Raspberry Pi Foundation","model":"Raspberry Pi 4 Model B","serial":"10000000abcdef01"},"cpu":{"architecture":"aarch64"},"os":{"distribution":"Red Hat Enterprise Linux","version":"9.3","fips_enabled":true}}`,
		},
		{
			name: "without TPM",
			input: protocol.SystemInfo{
				TPM: protocol.TPMInfo{
					Present:      false,
					Manufacturer: nil,
					Model:        nil,
					Version:      nil,
				},
				Board: protocol.BoardInfo{
					Manufacturer: "Dell Inc.",
					Model:        "OptiPlex 7090",
					Serial:       "ABC123DEF456",
				},
				CPU: protocol.CPUInfo{
					Architecture: "x86_64",
				},
				OS: protocol.OSInfo{
					Distribution: "Ubuntu",
					Version:      "22.04",
					FIPSEnabled:  false,
				},
			},
			expected: `{"tpm":{"present":false,"manufacturer":null,"model":null,"version":null},"board":{"manufacturer":"Dell Inc.","model":"OptiPlex 7090","serial":"ABC123DEF456"},"cpu":{"architecture":"x86_64"},"os":{"distribution":"Ubuntu","version":"22.04","fips_enabled":false}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.input)
			require.NoError(t, err)
			assert.JSONEq(t, tt.expected, string(data))

			var decoded protocol.SystemInfo
			err = json.Unmarshal(data, &decoded)
			require.NoError(t, err)
			assert.Equal(t, tt.input, decoded)
		})
	}
}

func TestNetworkConfig_JSON(t *testing.T) {
	tests := []struct {
		name     string
		input    protocol.NetworkConfig
		expected string
	}{
		{
			name: "multiple interfaces with addresses",
			input: protocol.NetworkConfig{
				Interfaces: []protocol.NetworkInterface{
					{
						Name:       "eth0",
						MACAddress: "dc:a6:32:12:34:56",
						LinkState:  "up",
						IPAddresses: []protocol.IPAddress{
							{
								IP:     "192.168.1.100",
								Prefix: 24,
								Family: "ipv4",
							},
							{
								IP:     "fe80::dea6:32ff:fe12:3456",
								Prefix: 64,
								Family: "ipv6",
							},
						},
					},
					{
						Name:        "wlan0",
						MACAddress:  "b8:27:eb:98:76:54",
						LinkState:   "down",
						IPAddresses: []protocol.IPAddress{},
					},
				},
			},
			expected: `{"interfaces":[{"name":"eth0","mac_address":"dc:a6:32:12:34:56","link_state":"up","ip_addresses":[{"ip":"192.168.1.100","prefix":24,"family":"ipv4"},{"ip":"fe80::dea6:32ff:fe12:3456","prefix":64,"family":"ipv6"}]},{"name":"wlan0","mac_address":"b8:27:eb:98:76:54","link_state":"down","ip_addresses":[]}]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.input)
			require.NoError(t, err)
			assert.JSONEq(t, tt.expected, string(data))

			var decoded protocol.NetworkConfig
			err = json.Unmarshal(data, &decoded)
			require.NoError(t, err)
			assert.Equal(t, tt.input, decoded)
		})
	}
}

func TestConfigBundle_JSON(t *testing.T) {
	tests := []struct {
		name     string
		input    protocol.ConfigBundle
		expected string
	}{
		{
			name: "multiple files",
			input: protocol.ConfigBundle{
				Files: []protocol.ConfigFile{
					{
						Path:    "systemd/network/10-eth0.network",
						Content: "W01hdGNoXQpOYW1lPWV0aDAK",
						Mode:    420,
					},
					{
						Path:    "chrony/chrony.conf",
						Content: "c2VydmVyIHRpbWUuY2xvdWRmbGFyZS5jb20K",
						Mode:    420,
					},
				},
			},
			expected: `{"files":[{"path":"systemd/network/10-eth0.network","content":"W01hdGNoXQpOYW1lPWV0aDAK","mode":420},{"path":"chrony/chrony.conf","content":"c2VydmVyIHRpbWUuY2xvdWRmbGFyZS5jb20K","mode":420}]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.input)
			require.NoError(t, err)
			assert.JSONEq(t, tt.expected, string(data))

			var decoded protocol.ConfigBundle
			err = json.Unmarshal(data, &decoded)
			require.NoError(t, err)
			assert.Equal(t, tt.input, decoded)
		})
	}
}

func TestCommandRequest_JSON(t *testing.T) {
	tests := []struct {
		name     string
		input    protocol.CommandRequest
		expected string
	}{
		{
			name:     "reboot command",
			input:    protocol.CommandRequest{ID: "reboot"},
			expected: `{"id":"reboot"}`,
		},
		{
			name:     "reload-networkd command",
			input:    protocol.CommandRequest{ID: "reload-networkd"},
			expected: `{"id":"reload-networkd"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.input)
			require.NoError(t, err)
			assert.JSONEq(t, tt.expected, string(data))

			var decoded protocol.CommandRequest
			err = json.Unmarshal(data, &decoded)
			require.NoError(t, err)
			assert.Equal(t, tt.input, decoded)
		})
	}
}

func TestCommandResponse_JSON(t *testing.T) {
	tests := []struct {
		name     string
		input    protocol.CommandResponse
		expected string
	}{
		{
			name: "success",
			input: protocol.CommandResponse{
				ExitCode: 0,
				Stdout:   "Service reloaded successfully.",
				Stderr:   "",
			},
			expected: `{"exit_code":0,"stdout":"Service reloaded successfully.","stderr":""}`,
		},
		{
			name: "failure",
			input: protocol.CommandResponse{
				ExitCode: 1,
				Stdout:   "",
				Stderr:   "Failed to reload service: Unit not found.",
			},
			expected: `{"exit_code":1,"stdout":"","stderr":"Failed to reload service: Unit not found."}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.input)
			require.NoError(t, err)
			assert.JSONEq(t, tt.expected, string(data))

			var decoded protocol.CommandResponse
			err = json.Unmarshal(data, &decoded)
			require.NoError(t, err)
			assert.Equal(t, tt.input, decoded)
		})
	}
}

func TestSRPRequests_JSON(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{
			name: "SRP init request",
			input: protocol.SRPInitRequest{
				Username: "boardingpass",
				A:        "dGVzdEFwaGVtZXJhbA==",
			},
			expected: `{"username":"boardingpass","A":"dGVzdEFwaGVtZXJhbA=="}`,
		},
		{
			name: "SRP init response",
			input: protocol.SRPInitResponse{
				Salt: "c29tZXJhbmRvbXNhbHQ=",
				B:    "dGVzdEJwaGVtZXJhbA==",
			},
			expected: `{"salt":"c29tZXJhbmRvbXNhbHQ=","b":"dGVzdEJwaGVtZXJhbA=="}`,
		},
		{
			name: "SRP verify request",
			input: protocol.SRPVerifyRequest{
				M1: "dGVzdE0xUHJvb2Y=",
			},
			expected: `{"M1":"dGVzdE0xUHJvb2Y="}`,
		},
		{
			name: "SRP verify response",
			input: protocol.SRPVerifyResponse{
				M2:           "dGVzdE0yUHJvb2Y=",
				SessionToken: "dG9rZW5faWQ.c2lnbmF0dXJl",
			},
			expected: `{"M2":"dGVzdE0yUHJvb2Y=","session_token":"dG9rZW5faWQ.c2lnbmF0dXJl"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.input)
			require.NoError(t, err)
			assert.JSONEq(t, tt.expected, string(data))
		})
	}
}

func TestCompleteResponse_JSON(t *testing.T) {
	t.Run("with message", func(t *testing.T) {
		input := protocol.CompleteResponse{
			Status:       "shutting_down",
			SentinelFile: "/etc/boardingpass/issued",
			Message:      stringPtr("Device provisioning complete"),
		}
		expected := `{"status":"shutting_down","sentinel_file":"/etc/boardingpass/issued","message":"Device provisioning complete"}`

		data, err := json.Marshal(input)
		require.NoError(t, err)
		assert.JSONEq(t, expected, string(data))

		var decoded protocol.CompleteResponse
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)
		assert.Equal(t, input.Status, decoded.Status)
		assert.Equal(t, input.SentinelFile, decoded.SentinelFile)
		require.NotNil(t, decoded.Message)
		assert.Equal(t, *input.Message, *decoded.Message)
	})

	t.Run("without message (omitempty)", func(t *testing.T) {
		input := protocol.CompleteResponse{
			Status:       "shutting_down",
			SentinelFile: "/etc/boardingpass/issued",
			Message:      nil,
		}
		expected := `{"status":"shutting_down","sentinel_file":"/etc/boardingpass/issued"}`

		data, err := json.Marshal(input)
		require.NoError(t, err)
		assert.JSONEq(t, expected, string(data))

		var decoded protocol.CompleteResponse
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)
		assert.Equal(t, input.Status, decoded.Status)
		assert.Equal(t, input.SentinelFile, decoded.SentinelFile)
		assert.Nil(t, decoded.Message)
	})
}

func stringPtr(s string) *string {
	return &s
}
