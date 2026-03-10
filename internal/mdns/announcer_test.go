package mdns

import (
	"net"
	"strings"
	"testing"

	"github.com/fzdarsky/boardingpass/internal/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testLogger() *logging.Logger {
	return logging.New(logging.LevelWarn, logging.FormatJSON)
}

func testRecord() ServiceRecord {
	return ServiceRecord{
		Instance: "BoardingPass-test",
		Service:  "_boardingpass._tcp",
		Domain:   "local",
		Port:     8443,
		TXT:      map[string]string{"version": "1.0"},
		Addrs:    []net.IP{net.IPv4(10, 0, 0, 1)},
	}
}

func TestServiceRecord_Names(t *testing.T) {
	r := testRecord()
	assert.Equal(t, "_boardingpass._tcp.local.", r.fqServiceName())
	assert.Equal(t, "BoardingPass-test._boardingpass._tcp.local.", r.fqInstanceName())
	assert.Equal(t, "BoardingPass-test.local.", r.fqHostName())
}

func TestBuildResponse(t *testing.T) {
	a := NewAnnouncer(testRecord(), testLogger())
	msg := a.buildResponse(120)

	// Should have PTR in answers
	require.Len(t, msg.Answers, 1)
	assert.Equal(t, TypePTR, msg.Answers[0].Type)
	assert.Equal(t, "_boardingpass._tcp.local.", msg.Answers[0].Name)
	assert.Equal(t, uint32(120), msg.Answers[0].TTL)
	assert.Equal(t, ClassIN, msg.Answers[0].Class) // PTR: no cache-flush

	// Should have SRV + TXT + A in additional
	require.Len(t, msg.Additional, 3)
	assert.Equal(t, TypeSRV, msg.Additional[0].Type)
	assert.Equal(t, TypeTXT, msg.Additional[1].Type)
	assert.Equal(t, TypeA, msg.Additional[2].Type)

	// SRV and TXT should use instance name
	assert.Equal(t, "BoardingPass-test._boardingpass._tcp.local.", msg.Additional[0].Name)
	assert.Equal(t, "BoardingPass-test._boardingpass._tcp.local.", msg.Additional[1].Name)

	// A record should use host name
	assert.Equal(t, "BoardingPass-test.local.", msg.Additional[2].Name)
	assert.Equal(t, []byte{10, 0, 0, 1}, msg.Additional[2].Data)

	// SRV, TXT, A should have cache-flush bit set
	assert.Equal(t, ClassINFlush, msg.Additional[0].Class)
	assert.Equal(t, ClassINFlush, msg.Additional[1].Class)
	assert.Equal(t, ClassINFlush, msg.Additional[2].Class)

	// Header should be response + authoritative
	assert.Equal(t, flagQR|flagAA, msg.Header.Flags)
}

func TestBuildResponse_Goodbye(t *testing.T) {
	a := NewAnnouncer(testRecord(), testLogger())
	msg := a.buildResponse(0)

	// All records should have TTL=0
	assert.Equal(t, uint32(0), msg.Answers[0].TTL)
	for _, rr := range msg.Additional {
		assert.Equal(t, uint32(0), rr.TTL)
	}
}

func TestBuildResponse_MultipleAddresses(t *testing.T) {
	r := testRecord()
	r.Addrs = []net.IP{net.IPv4(10, 0, 0, 1), net.IPv4(10, 0, 1, 1)}
	a := NewAnnouncer(r, testLogger())
	msg := a.buildResponse(120)

	// Should have SRV + TXT + 2x A
	require.Len(t, msg.Additional, 4)
	assert.Equal(t, TypeA, msg.Additional[2].Type)
	assert.Equal(t, TypeA, msg.Additional[3].Type)
	assert.Equal(t, []byte{10, 0, 0, 1}, msg.Additional[2].Data)
	assert.Equal(t, []byte{10, 0, 1, 1}, msg.Additional[3].Data)
}

func TestBuildResponse_PackableMessage(t *testing.T) {
	a := NewAnnouncer(testRecord(), testLogger())
	msg := a.buildResponse(120)

	// Verify the response can be serialized
	data, err := PackMessage(msg)
	require.NoError(t, err)
	assert.True(t, len(data) > headerSize)

	// Verify it can be deserialized back
	parsed, err := UnpackMessage(data)
	require.NoError(t, err)
	assert.Len(t, parsed.Answers, 1)
	assert.Len(t, parsed.Additional, 3)
}

func TestMatchesQuestion(t *testing.T) {
	a := NewAnnouncer(testRecord(), testLogger())

	tests := []struct {
		name    string
		qName   string
		qType   uint16
		matches bool
	}{
		{"PTR for service", "_boardingpass._tcp.local.", TypePTR, true},
		{"SRV for instance", "BoardingPass-test._boardingpass._tcp.local.", TypeSRV, true},
		{"TXT for instance", "BoardingPass-test._boardingpass._tcp.local.", TypeTXT, true},
		{"A for host", "BoardingPass-test.local.", TypeA, true},
		{"DNS-SD browse", "_services._dns-sd._udp.local.", TypePTR, true},
		{"case insensitive", "_BOARDINGPASS._TCP.LOCAL.", TypePTR, true},
		{"wrong service", "_other._tcp.local.", TypePTR, false},
		{"wrong type for service", "_boardingpass._tcp.local.", TypeA, false},
		{"wrong host", "other.local.", TypeA, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := a.matchesQuestion(strings.ToLower(tt.qName), tt.qType)
			assert.Equal(t, tt.matches, got)
		})
	}
}

func TestAddRemoveAddress(t *testing.T) {
	a := NewAnnouncer(testRecord(), testLogger())

	// Add new address
	a.AddAddress(net.IPv4(10, 0, 1, 1))
	a.mu.RLock()
	assert.Len(t, a.record.Addrs, 2)
	a.mu.RUnlock()

	// Add duplicate — should not change
	a.AddAddress(net.IPv4(10, 0, 1, 1))
	a.mu.RLock()
	assert.Len(t, a.record.Addrs, 2)
	a.mu.RUnlock()

	// Remove address
	a.RemoveAddress(net.IPv4(10, 0, 0, 1))
	a.mu.RLock()
	assert.Len(t, a.record.Addrs, 1)
	assert.True(t, a.record.Addrs[0].Equal(net.IPv4(10, 0, 1, 1)))
	a.mu.RUnlock()

	// Ignore IPv6
	a.AddAddress(net.ParseIP("::1"))
	a.mu.RLock()
	assert.Len(t, a.record.Addrs, 1)
	a.mu.RUnlock()
}

func TestIPStrings(t *testing.T) {
	ips := []net.IP{net.IPv4(10, 0, 0, 1), net.IPv4(192, 168, 1, 1)}
	s := ipStrings(ips)
	assert.Equal(t, []string{"10.0.0.1", "192.168.1.1"}, s)
}
