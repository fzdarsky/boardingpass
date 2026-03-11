package integration_test

import (
	"net"
	"testing"
	"time"

	"github.com/fzdarsky/boardingpass/internal/logging"
	"github.com/fzdarsky/boardingpass/internal/mdns"
	"github.com/stretchr/testify/require"
)

func TestMDNSAnnouncer_StartStop(t *testing.T) {
	logger := logging.New(logging.LevelWarn, logging.FormatJSON)
	record := mdns.ServiceRecord{
		Instance: "BoardingPass-test",
		Service:  "_boardingpass._tcp",
		Domain:   "local",
		Port:     9455,
		TXT:      map[string]string{"version": "test"},
		Addrs:    []net.IP{net.IPv4(127, 0, 0, 1)},
	}

	announcer := mdns.NewAnnouncer(record, logger)

	err := announcer.Start(t.Context())
	require.NoError(t, err)

	// Give the announcer time to send initial announcements
	time.Sleep(100 * time.Millisecond)

	announcer.Stop()
}
