package transport

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/fzdarsky/boardingpass/internal/config"
	"github.com/fzdarsky/boardingpass/internal/logging"
)

const bleEnvFile = "/run/boardingpass/ble-env"

// BluetoothHandler manages the Bluetooth PAN and BLE advertisement transports via systemd.
type BluetoothHandler struct {
	cfg    config.BluetoothTransport
	port   int
	logger *logging.Logger
	state  State
	mu     sync.Mutex
}

// NewBluetoothHandler creates a new Bluetooth transport handler.
func NewBluetoothHandler(cfg config.BluetoothTransport, port int, logger *logging.Logger) *BluetoothHandler {
	return &BluetoothHandler{
		cfg:    cfg,
		port:   port,
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

	// Write environment file before starting units — both bt@ and ble@ read it
	if err := b.writeBLEEnvFile(); err != nil {
		b.logger.Warn("failed to write env file", map[string]any{
			"error": err.Error(),
		})
	}

	// Start Bluetooth PAN unit — failure is non-fatal so BLE discovery still works.
	// On FIPS systems, PAN pairing fails (no ecdh_generic) but BLE advertisement
	// can still direct phones to connect via WiFi AP or Ethernet.
	btUnit := fmt.Sprintf("boardingpass-bt@%s", adapter)
	//nolint:gosec // G204: adapter name is validated from config
	cmd := exec.CommandContext(ctx, "sudo", "systemctl", "start", btUnit)
	if out, err := cmd.CombinedOutput(); err != nil {
		b.logger.Warn("Bluetooth PAN failed to start (BLE discovery still active)", map[string]any{
			"unit":  btUnit,
			"error": err.Error(),
			"out":   string(out),
		})
	}

	// Start BLE advertisement unit
	bleUnit := fmt.Sprintf("boardingpass-ble@%s", adapter)
	//nolint:gosec // G204: adapter name is validated from config
	cmd = exec.CommandContext(ctx, "sudo", "systemctl", "start", bleUnit)
	if out, err := cmd.CombinedOutput(); err != nil {
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

	// Stop Bluetooth PAN (may not be running if it failed to start)
	btUnit := fmt.Sprintf("boardingpass-bt@%s", adapter)
	//nolint:gosec // G204: adapter name is validated from config
	cmd = exec.CommandContext(ctx, "sudo", "systemctl", "stop", btUnit)
	if out, err := cmd.CombinedOutput(); err != nil {
		b.logger.Warn("failed to stop Bluetooth PAN", map[string]any{
			"unit":  btUnit,
			"error": err.Error(),
			"out":   string(out),
		})
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

// writeBLEEnvFile writes an environment file for the BLE advertisement script.
func (b *BluetoothHandler) writeBLEEnvFile() error {
	dir := filepath.Dir(bleEnvFile)
	//nolint:gosec // G301: /run/boardingpass is a standard runtime directory
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	deviceName := b.cfg.DeviceName
	if deviceName == "" {
		hostname, _ := os.Hostname()
		deviceName = "BoardingPass-" + hostname
	}

	content := fmt.Sprintf("BP_DEVICE_NAME=%s\nBP_ADDRESS=%s\nBP_PORT=%d\n",
		deviceName, b.cfg.Address, b.port)

	//nolint:gosec // G306: env file does not contain secrets
	if err := os.WriteFile(bleEnvFile, []byte(content), 0o644); err != nil {
		return fmt.Errorf("failed to write BLE env file: %w", err)
	}

	return nil
}
