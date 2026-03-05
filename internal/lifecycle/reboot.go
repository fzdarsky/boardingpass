package lifecycle

import (
	"context"
	"log"
	"os/exec"
)

// SystemReboot triggers a system reboot via systemctl.
// This function is intended to be called after the HTTP response has been sent
// and a short delay has elapsed.
func SystemReboot() {
	//nolint:gosec // G204: reboot command path is hardcoded, not user-controlled
	cmd := exec.CommandContext(context.Background(), "/usr/bin/systemctl", "reboot", "--force")
	if err := cmd.Run(); err != nil {
		log.Printf("ERROR: failed to execute reboot: %v", err)
	}
}
