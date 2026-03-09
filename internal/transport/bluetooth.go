package transport

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"

	"github.com/fzdarsky/boardingpass/internal/config"
	"github.com/fzdarsky/boardingpass/internal/logging"
)

// BluetoothHandler manages the Bluetooth PAN and BLE advertisement transports via systemd.
type BluetoothHandler struct {
	cfg    config.BluetoothTransport
	logger *logging.Logger
	state  State
	mu     sync.Mutex
}

// NewBluetoothHandler creates a new Bluetooth transport handler.
func NewBluetoothHandler(cfg config.BluetoothTransport, logger *logging.Logger) *BluetoothHandler {
	return &BluetoothHandler{
		cfg:    cfg,
		logger: logger,
		state:  StateDisabled,
	}
}

// Start activates the Bluetooth PAN and BLE advertisement systemd units.
func (b *BluetoothHandler) Start(ctx context.Context) error {
	b.mu.Lock()
	b.state = StateStarting
	b.mu.Unlock()

	adapter := b.cfg.Adapter
	if adapter == "" {
		adapter = "hci0"
	}

	// Validate adapter exists
	if _, err := os.Stat(fmt.Sprintf("/sys/class/bluetooth/%s", adapter)); err != nil {
		b.setState(StateFailed)
		return fmt.Errorf("bluetooth adapter %s not found: %w", adapter, err)
	}

	// Start Bluetooth PAN unit
	btUnit := fmt.Sprintf("boardingpass-bt@%s", adapter)
	//nolint:gosec // G204: adapter name is validated from config
	cmd := exec.CommandContext(ctx, "sudo", "systemctl", "start", btUnit)
	if out, err := cmd.CombinedOutput(); err != nil {
		b.setState(StateFailed)
		return fmt.Errorf("failed to start %s: %s: %w", btUnit, string(out), err)
	}

	// Start BLE advertisement unit
	bleUnit := fmt.Sprintf("boardingpass-ble@%s", adapter)
	//nolint:gosec // G204: adapter name is validated from config
	cmd = exec.CommandContext(ctx, "sudo", "systemctl", "start", bleUnit)
	if out, err := cmd.CombinedOutput(); err != nil {
		// BLE failure is non-fatal — PAN is still functional
		b.logger.Warn("BLE advertisement failed to start", map[string]any{
			"unit":  bleUnit,
			"error": err.Error(),
			"out":   string(out),
		})
	}

	b.setState(StateActive)
	return nil
}

// Stop deactivates the Bluetooth PAN and BLE advertisement systemd units.
func (b *BluetoothHandler) Stop(ctx context.Context) error {
	b.mu.Lock()
	b.state = StateStopping
	b.mu.Unlock()

	adapter := b.cfg.Adapter
	if adapter == "" {
		adapter = "hci0"
	}

	// Stop BLE advertisement first
	bleUnit := fmt.Sprintf("boardingpass-ble@%s", adapter)
	//nolint:gosec // G204: adapter name is validated from config
	cmd := exec.CommandContext(ctx, "sudo", "systemctl", "stop", bleUnit)
	if out, err := cmd.CombinedOutput(); err != nil {
		b.logger.Warn("failed to stop BLE advertisement", map[string]any{
			"unit":  bleUnit,
			"error": err.Error(),
			"out":   string(out),
		})
	}

	// Stop Bluetooth PAN
	btUnit := fmt.Sprintf("boardingpass-bt@%s", adapter)
	//nolint:gosec // G204: adapter name is validated from config
	cmd = exec.CommandContext(ctx, "sudo", "systemctl", "stop", btUnit)
	if out, err := cmd.CombinedOutput(); err != nil {
		b.setState(StateFailed)
		return fmt.Errorf("failed to stop %s: %s: %w", btUnit, string(out), err)
	}

	b.setState(StateStopped)
	return nil
}

// TransportType returns the transport type.
func (b *BluetoothHandler) TransportType() Type {
	return TypeBluetooth
}

// TransportState returns the current state.
func (b *BluetoothHandler) TransportState() State {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.state
}

func (b *BluetoothHandler) setState(s State) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.state = s
}
