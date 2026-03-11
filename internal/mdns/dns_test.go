package mdns

import (
	"encoding/binary"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []byte
		wantErr  bool
	}{
		{
			name:     "simple name",
			input:    "local.",
			expected: []byte{5, 'l', 'o', 'c', 'a', 'l', 0},
		},
		{
			name:     "no trailing dot",
			input:    "local",
			expected: []byte{5, 'l', 'o', 'c', 'a', 'l', 0},
		},
		{
			name:     "multi-label",
			input:    "_boardingpass._tcp.local.",
			expected: append(append(append([]byte{13}, "_boardingpass"...), append([]byte{4}, "_tcp"...)...), append([]byte{5}, "local"...)...),
		},
		{
			name:     "root",
			input:    "",
			expected: []byte{0},
		},
		{
			name:    "empty label",
			input:   "a..b",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := encodeName(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			// Append root terminator to multi-label expected
			if tt.name == "multi-label" {
				tt.expected = append(tt.expected, 0)
			}
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestDecodeName(t *testing.T) {
	tests := []struct {
		name       string
		data       []byte
		off        int
		wantName   string
		wantNewOff int
		wantErr    bool
	}{
		{
			name:       "simple name",
			data:       []byte{5, 'l', 'o', 'c', 'a', 'l', 0},
			off:        0,
			wantName:   "local.",
			wantNewOff: 7,
		},
		{
			name: "with compression pointer",
			// Offset 0: "local\0" (7 bytes), offset 7: pointer to 0
			data:       append([]byte{5, 'l', 'o', 'c', 'a', 'l', 0}, 0xC0, 0x00),
			off:        7,
			wantName:   "local.",
			wantNewOff: 9,
		},
		{
			name: "label then pointer",
			// Offset 0: "local\0", offset 7: "sub" + pointer to 0
			data: append(
				[]byte{5, 'l', 'o', 'c', 'a', 'l', 0},
				3, 's', 'u', 'b', 0xC0, 0x00,
			),
			off:        7,
			wantName:   "sub.local.",
			wantNewOff: 13,
		},
		{
			name:    "truncated",
			data:    []byte{5, 'l', 'o'},
			off:     0,
			wantErr: true,
		},
		{
			name:    "pointer loop",
			data:    []byte{0xC0, 0x00},
			off:     0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, newOff, err := decodeName(tt.data, tt.off)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantName, name)
			assert.Equal(t, tt.wantNewOff, newOff)
		})
	}
}

func TestPackUnpackRoundTrip(t *testing.T) {
	original := &Message{
		Header: Header{
			ID:    0,
			Flags: flagQR | flagAA,
		},
		Questions: []Question{
			{Name: "_boardingpass._tcp.local.", Type: TypePTR, Class: ClassIN},
		},
		Answers: []ResourceRecord{
			{
				Name:  "_boardingpass._tcp.local.",
				Type:  TypePTR,
				Class: ClassIN,
				TTL:   120,
				Data:  mustEncodeName(t, "MyDevice._boardingpass._tcp.local."),
			},
		},
		Additional: []ResourceRecord{
			{
				Name:  "MyDevice.local.",
				Type:  TypeA,
				Class: ClassINFlush,
				TTL:   120,
				Data:  NewARecord(net.IPv4(10, 0, 0, 1)),
			},
		},
	}

	packed, err := PackMessage(original)
	require.NoError(t, err)

	unpacked, err := UnpackMessage(packed)
	require.NoError(t, err)

	assert.Equal(t, original.Header.Flags, unpacked.Header.Flags)
	assert.Len(t, unpacked.Questions, 1)
	assert.Equal(t, "_boardingpass._tcp.local.", unpacked.Questions[0].Name)
	assert.Equal(t, TypePTR, unpacked.Questions[0].Type)
	assert.Len(t, unpacked.Answers, 1)
	assert.Equal(t, TypePTR, unpacked.Answers[0].Type)
	assert.Equal(t, uint32(120), unpacked.Answers[0].TTL)
	assert.Len(t, unpacked.Additional, 1)
	assert.Equal(t, TypeA, unpacked.Additional[0].Type)
	assert.Equal(t, []byte{10, 0, 0, 1}, unpacked.Additional[0].Data)
}

func TestNewARecord(t *testing.T) {
	tests := []struct {
		name     string
		ip       net.IP
		expected []byte
	}{
		{"IPv4", net.IPv4(192, 168, 1, 1), []byte{192, 168, 1, 1}},
		{"IPv6 returns nil", net.ParseIP("::1"), nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, NewARecord(tt.ip))
		})
	}
}

func TestNewSRVRecord(t *testing.T) {
	data, err := NewSRVRecord(0, 0, 9455, "myhost.local.")
	require.NoError(t, err)

	assert.Equal(t, uint16(0), binary.BigEndian.Uint16(data[0:2]))    // priority
	assert.Equal(t, uint16(0), binary.BigEndian.Uint16(data[2:4]))    // weight
	assert.Equal(t, uint16(9455), binary.BigEndian.Uint16(data[4:6])) // port

	// Target should be encoded name
	name, _, err := decodeName(data[6:], 0)
	require.NoError(t, err)
	assert.Equal(t, "myhost.local.", name)
}

func TestNewTXTRecord(t *testing.T) {
	tests := []struct {
		name     string
		kv       map[string]string
		validate func(t *testing.T, data []byte)
	}{
		{
			name: "single pair",
			kv:   map[string]string{"version": "1.0"},
			validate: func(t *testing.T, data []byte) {
				require.True(t, len(data) > 1)
				strLen := int(data[0])
				assert.Equal(t, "version=1.0", string(data[1:1+strLen]))
			},
		},
		{
			name: "empty map",
			kv:   map[string]string{},
			validate: func(t *testing.T, data []byte) {
				assert.Equal(t, []byte{0}, data)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := NewTXTRecord(tt.kv)
			tt.validate(t, data)
		})
	}
}

func TestNewPTRRecord(t *testing.T) {
	data, err := NewPTRRecord("MyDevice._boardingpass._tcp.local.")
	require.NoError(t, err)

	name, _, err := decodeName(data, 0)
	require.NoError(t, err)
	assert.Equal(t, "MyDevice._boardingpass._tcp.local.", name)
}

func TestUnpackMessage_TooShort(t *testing.T) {
	_, err := UnpackMessage([]byte{0, 1, 2})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "too short")
}

func mustEncodeName(t *testing.T, name string) []byte {
	t.Helper()
	data, err := encodeName(name)
	require.NoError(t, err)
	return data
}
