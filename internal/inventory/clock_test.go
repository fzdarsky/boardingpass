package inventory

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTimedatectlOutput(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		wantTime time.Time
		wantSync bool
		wantErr  bool
	}{
		{
			name:     "synchronized",
			output:   "NTPSynchronized=yes\nTimeUSec=Thu 2026-03-05 14:30:00 UTC\n",
			wantTime: time.Date(2026, 3, 5, 14, 30, 0, 0, time.UTC),
			wantSync: true,
		},
		{
			name:     "not synchronized",
			output:   "NTPSynchronized=no\nTimeUSec=Thu 2026-03-05 14:30:00 UTC\n",
			wantTime: time.Date(2026, 3, 5, 14, 30, 0, 0, time.UTC),
			wantSync: false,
		},
		{
			name:     "reversed order",
			output:   "TimeUSec=Thu 2026-03-05 14:30:00 UTC\nNTPSynchronized=yes\n",
			wantTime: time.Date(2026, 3, 5, 14, 30, 0, 0, time.UTC),
			wantSync: true,
		},
		{
			name:     "extra whitespace",
			output:   "  NTPSynchronized=yes  \n  TimeUSec=Thu 2026-03-05 14:30:00 UTC  \n",
			wantTime: time.Date(2026, 3, 5, 14, 30, 0, 0, time.UTC),
			wantSync: true,
		},
		{
			name:    "missing NTPSynchronized",
			output:  "TimeUSec=Thu 2026-03-05 14:30:00 UTC\n",
			wantErr: true,
		},
		{
			name:    "missing TimeUSec",
			output:  "NTPSynchronized=yes\n",
			wantErr: true,
		},
		{
			name:    "malformed time",
			output:  "NTPSynchronized=yes\nTimeUSec=not-a-date\n",
			wantErr: true,
		},
		{
			name:    "empty output",
			output:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTime, gotSync, err := parseTimedatectlOutput(tt.output)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantTime, gotTime)
			assert.Equal(t, tt.wantSync, gotSync)
		})
	}
}

func TestParseTimedatectlTime(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    time.Time
		wantErr bool
	}{
		{
			name:  "standard format",
			input: "Thu 2026-03-05 14:30:00 UTC",
			want:  time.Date(2026, 3, 5, 14, 30, 0, 0, time.UTC),
		},
		{
			name:  "midnight",
			input: "Mon 2026-01-01 00:00:00 UTC",
			want:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:    "invalid format",
			input:   "2026-03-05T14:30:00Z",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTimedatectlTime(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
