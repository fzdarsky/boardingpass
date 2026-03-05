package inventory_test

import (
	"testing"

	"github.com/fzdarsky/boardingpass/internal/inventory"
	"github.com/stretchr/testify/assert"
)

func TestGetTPMInfo(t *testing.T) {
	// Basic smoke test - should not panic
	info, err := inventory.GetTPMInfo()

	assert.NoError(t, err, "GetTPMInfo should not return an error")

	// TPM may or may not be present on test system
	if info.Present {
		// If TPM is present, manufacturer, model, and version may be populated
		t.Logf("TPM detected: Manufacturer=%v, Model=%v, Version=%v",
			ptrToString(info.Manufacturer),
			ptrToString(info.Model),
			ptrToString(info.Version))
	} else {
		// If no TPM, manufacturer, model, and version should be nil
		assert.Nil(t, info.Manufacturer, "Manufacturer should be nil when TPM not present")
		assert.Nil(t, info.Model, "Model should be nil when TPM not present")
		assert.Nil(t, info.Version, "Version should be nil when TPM not present")
	}
}

func TestTranslateManufacturerID(t *testing.T) {
	tests := []struct {
		name   string
		hexStr string
		want   string
	}{
		{
			name:   "STMicroelectronics",
			hexStr: "53544d20", // "STM " in hex
			want:   "STMicroelectronics",
		},
		{
			name:   "Microsoft",
			hexStr: "4d534654", // "MSFT" in hex
			want:   "Microsoft",
		},
		{
			name:   "Intel",
			hexStr: "494e5443", // "INTC" in hex
			want:   "Intel",
		},
		{
			name:   "Infineon",
			hexStr: "49465800", // "IFX\x00" in hex
			want:   "Infineon",
		},
		{
			name:   "AMD",
			hexStr: "414d4400", // "AMD\x00" in hex
			want:   "AMD",
		},
		{
			name:   "Nuvoton",
			hexStr: "4e544300", // "NTC\x00" in hex
			want:   "Nuvoton",
		},
		{
			name:   "unknown manufacturer returns ASCII",
			hexStr: "58595a41", // "XYZA" in hex
			want:   "XYZA",
		},
		{
			name:   "invalid hex length returns empty",
			hexStr: "1234",
			want:   "",
		},
		{
			name:   "invalid hex characters returns empty",
			hexStr: "GGHHIIJJ",
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := inventory.TranslateManufacturerID(tt.hexStr)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseManufacturerFromCaps(t *testing.T) {
	tests := []struct {
		name string
		caps string
		want string
	}{
		{
			name: "STMicroelectronics from caps",
			caps: "Manufacturer: 0x53544d20\nTCG version: 1.2\n",
			want: "STMicroelectronics",
		},
		{
			name: "Microsoft from caps",
			caps: "Manufacturer: 0x4d534654\nFirmware version: 1.0\n",
			want: "Microsoft",
		},
		{
			name: "no manufacturer line",
			caps: "TCG version: 2.0\n",
			want: "",
		},
		{
			name: "empty caps",
			caps: "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := inventory.ParseManufacturerFromCaps(tt.caps)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseManufacturerFromModalias(t *testing.T) {
	tests := []struct {
		name     string
		modalias string
		want     string
	}{
		{
			name:     "Microsoft from MSFT",
			modalias: "acpi:MSFT0101:",
			want:     "Microsoft",
		},
		{
			name:     "Intel from INTC",
			modalias: "acpi:INTC0102:",
			want:     "Intel",
		},
		{
			name:     "AMD from AMD_",
			modalias: "acpi:AMD_0020:",
			want:     "AMD",
		},
		{
			name:     "unknown vendor returns empty",
			modalias: "acpi:UNKN0001:",
			want:     "",
		},
		{
			name:     "platform format returns empty",
			modalias: "platform:tpm_crb:",
			want:     "",
		},
		{
			name:     "empty modalias",
			modalias: "",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := inventory.ParseManufacturerFromModalias(tt.modalias)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseModelFromModalias(t *testing.T) {
	tests := []struct {
		name     string
		modalias string
		want     string
	}{
		{
			name:     "Microsoft firmware TPM",
			modalias: "acpi:MSFT0101:",
			want:     "Firmware TPM (fTPM)",
		},
		{
			name:     "Intel PTT",
			modalias: "acpi:INTC0102:",
			want:     "Intel Platform Trust Technology (PTT)",
		},
		{
			name:     "AMD fTPM",
			modalias: "acpi:AMD0020:",
			want:     "AMD Platform Security Processor (fTPM)",
		},
		{
			name:     "platform CRB",
			modalias: "platform:tpm_crb:",
			want:     "Command Response Buffer (CRB)",
		},
		{
			name:     "unknown acpi device",
			modalias: "acpi:VENDOR0001:",
			want:     "VENDOR0001",
		},
		{
			name:     "empty modalias",
			modalias: "",
			want:     "",
		},
		{
			name:     "unknown format",
			modalias: "pci:v00001234d00005678",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := inventory.ParseModelFromModalias(tt.modalias)
			assert.Equal(t, tt.want, got)
		})
	}
}

func ptrToString(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}
