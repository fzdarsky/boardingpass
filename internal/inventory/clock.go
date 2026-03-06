package inventory

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// GetClockStatus detects system time and NTP synchronization status
// by parsing timedatectl show output. Falls back to current time and
// unsynchronized if timedatectl is unavailable.
func GetClockStatus() (time.Time, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx, "timedatectl", "show",
		"--property=NTPSynchronized",
		"--property=TimeUSec",
	).Output()
	if err != nil {
		// timedatectl unavailable — fallback to current time, not synchronized
		return time.Now().UTC(), false, nil
	}

	return parseTimedatectlOutput(string(out))
}

// parseTimedatectlOutput extracts clock status from timedatectl show output.
// Expected format:
//
//	NTPSynchronized=yes
//	TimeUSec=Thu 2026-03-05 14:30:00 UTC
func parseTimedatectlOutput(output string) (time.Time, bool, error) {
	var synchronized bool
	var systemTime time.Time
	var foundTime, foundSync bool

	for rawLine := range strings.SplitSeq(output, "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "NTPSynchronized":
			synchronized = value == "yes"
			foundSync = true
		case "TimeUSec":
			t, err := parseTimedatectlTime(value)
			if err != nil {
				return time.Now().UTC(), false, fmt.Errorf("failed to parse TimeUSec %q: %w", value, err)
			}
			systemTime = t
			foundTime = true
		}
	}

	if !foundTime || !foundSync {
		return time.Now().UTC(), false, fmt.Errorf("incomplete timedatectl output: time=%v sync=%v", foundTime, foundSync)
	}

	return systemTime, synchronized, nil
}

// parseTimedatectlTime parses the timedatectl TimeUSec format.
// Example: "Thu 2026-03-05 14:30:00 UTC"
func parseTimedatectlTime(value string) (time.Time, error) {
	// timedatectl show outputs: "Thu 2026-03-05 14:30:00 UTC"
	return time.Parse("Mon 2006-01-02 15:04:05 MST", value)
}
