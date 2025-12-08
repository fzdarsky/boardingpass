package network_test

import (
	"net"
	"testing"

	"github.com/fzdarsky/boardingpass/internal/network"
	"github.com/stretchr/testify/assert"
)

func TestGetLinkState(t *testing.T) {
	tests := []struct {
		name     string
		flags    net.Flags
		expected string
	}{
		{
			name:     "interface up",
			flags:    net.FlagUp,
			expected: "up",
		},
		{
			name:     "interface down",
			flags:    0,
			expected: "down",
		},
		{
			name:     "interface up and broadcast",
			flags:    net.FlagUp | net.FlagBroadcast,
			expected: "up",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iface := net.Interface{
				Flags: tt.flags,
			}

			state := network.GetLinkState(iface)
			assert.Equal(t, tt.expected, state)
		})
	}
}
