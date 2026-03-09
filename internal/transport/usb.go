package transport

import (
	"context"
	"fmt"
	"maps"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fzdarsky/boardingpass/internal/config"
	"github.com/fzdarsky/boardingpass/internal/logging"
)

const (
	sysClassNet  = "/sys/class/net"
	pollInterval = 2 * time.Second
)

// ListenerCallback is called when a USB interface is detected with an IP address.
type ListenerCallback func(address string, port int) error

// ListenerRemoveCallback is called when a USB interface disappears.
type ListenerRemoveCallback func(address string) error

// USBHandler manages USB tethering interface detection via polling.
type USBHandler struct {
	cfg              config.USBTransport
	port             int
	logger           *logging.Logger
	state            State
	mu               sync.Mutex
	cancel           context.CancelFunc
	knownInterfaces  map[string]string // iface name -> bound address
	onListenerAdd    ListenerCallback
	onListenerRemove ListenerRemoveCallback
}

// NewUSBHandler creates a new USB transport handler.
func NewUSBHandler(cfg config.USBTransport, port int, logger *logging.Logger) *USBHandler {
	return &USBHandler{
		cfg:             cfg,
		port:            port,
		logger:          logger,
		state:           StateDisabled,
		knownInterfaces: make(map[string]string),
	}
}

// SetListenerCallbacks registers callbacks for dynamic listener management.
func (u *USBHandler) SetListenerCallbacks(add ListenerCallback, remove ListenerRemoveCallback) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.onListenerAdd = add
	u.onListenerRemove = remove
}

// Start begins polling for USB tethering interfaces.
func (u *USBHandler) Start(ctx context.Context) error {
	u.mu.Lock()
	u.state = StateStarting
	u.mu.Unlock()

	pollCtx, cancel := context.WithCancel(ctx)
	u.mu.Lock()
	u.cancel = cancel
	u.mu.Unlock()

	// Do an initial scan
	u.scanInterfaces()

	u.setState(StateActive)

	// Start polling in background
	go u.pollLoop(pollCtx)

	return nil
}

// Stop cancels the polling loop and cleans up known interfaces.
func (u *USBHandler) Stop(_ context.Context) error {
	u.mu.Lock()
	u.state = StateStopping
	cancel := u.cancel
	u.mu.Unlock()

	if cancel != nil {
		cancel()
	}

	// Remove listeners for all known interfaces
	u.mu.Lock()
	removeCallback := u.onListenerRemove
	interfaces := make(map[string]string, len(u.knownInterfaces))
	maps.Copy(interfaces, u.knownInterfaces)
	u.mu.Unlock()

	for iface, addr := range interfaces {
		if removeCallback != nil {
			if err := removeCallback(addr); err != nil {
				u.logger.Warn("failed to remove USB listener", map[string]any{
					"interface": iface,
					"address":   addr,
					"error":     err.Error(),
				})
			}
		}
	}

	u.mu.Lock()
	u.knownInterfaces = make(map[string]string)
	u.mu.Unlock()

	u.setState(StateStopped)
	return nil
}

// TransportType returns the transport type.
func (u *USBHandler) TransportType() Type {
	return TypeUSB
}

// TransportState returns the current state.
func (u *USBHandler) TransportState() State {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.state
}

func (u *USBHandler) setState(s State) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.state = s
}

func (u *USBHandler) pollLoop(ctx context.Context) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			u.scanInterfaces()
		}
	}
}

func (u *USBHandler) scanInterfaces() {
	// Read /sys/class/net/ for matching interfaces
	entries, err := os.ReadDir(sysClassNet)
	if err != nil {
		u.logger.Warn("failed to read /sys/class/net", map[string]any{
			"error": err.Error(),
		})
		return
	}

	currentIfaces := make(map[string]bool)

	for _, entry := range entries {
		name := entry.Name()

		// Primary detection: check if interface is backed by a known USB driver.
		// This works regardless of naming scheme (usb0, rndis0, enp0s20f0u1c4i2, etc.)
		if !u.isUSBInterface(name) {
			continue
		}

		// Optional prefix filter: if configured, restrict to matching interfaces
		if u.cfg.InterfacePrefix != "" && !u.matchesPrefix(name) {
			continue
		}

		currentIfaces[name] = true

		// Skip already-known interfaces
		u.mu.Lock()
		_, known := u.knownInterfaces[name]
		u.mu.Unlock()
		if known {
			continue
		}

		// Get IP address of the interface
		addr := u.getInterfaceAddress(name)
		if addr == "" {
			continue
		}

		bindAddr := fmt.Sprintf("%s:%d", addr, u.port)

		u.mu.Lock()
		addCallback := u.onListenerAdd
		u.mu.Unlock()

		if addCallback != nil {
			if err := addCallback(addr, u.port); err != nil {
				if strings.Contains(err.Error(), "address already in use") {
					// Another transport's listener (e.g., wildcard) already covers this address.
					// Track the interface so we don't retry, but no dedicated listener needed.
					u.mu.Lock()
					u.knownInterfaces[name] = bindAddr
					u.mu.Unlock()

					u.logger.Info("USB tethering interface detected (covered by existing listener)", map[string]any{
						"interface": name,
						"address":   bindAddr,
					})
					continue
				}
				u.logger.Warn("failed to add USB listener", map[string]any{
					"interface": name,
					"address":   bindAddr,
					"error":     err.Error(),
				})
				continue
			}
		}

		u.mu.Lock()
		u.knownInterfaces[name] = bindAddr
		u.mu.Unlock()

		u.logger.Info("USB tethering interface detected", map[string]any{
			"interface": name,
			"address":   bindAddr,
		})
	}

	// Detect disappeared interfaces
	u.mu.Lock()
	removeCallback := u.onListenerRemove
	disappeared := make(map[string]string)
	for iface, addr := range u.knownInterfaces {
		if !currentIfaces[iface] {
			disappeared[iface] = addr
		}
	}
	for iface := range disappeared {
		delete(u.knownInterfaces, iface)
	}
	u.mu.Unlock()

	for iface, addr := range disappeared {
		u.logger.Info("USB tethering interface disappeared", map[string]any{
			"interface": iface,
			"address":   addr,
		})

		if removeCallback != nil {
			if err := removeCallback(addr); err != nil {
				u.logger.Warn("failed to remove USB listener", map[string]any{
					"interface": iface,
					"address":   addr,
					"error":     err.Error(),
				})
			}
		}
	}
}

func (u *USBHandler) matchesPrefix(name string) bool {
	return strings.HasPrefix(name, u.cfg.InterfacePrefix)
}

func (u *USBHandler) isUSBInterface(name string) bool {
	// Check if the interface is backed by a USB driver
	driverLink := filepath.Join(sysClassNet, name, "device", "driver")
	target, err := os.Readlink(driverLink)
	if err != nil {
		return false
	}
	driver := filepath.Base(target)
	switch driver {
	case "cdc_ether", "rndis_host", "ipheth":
		return true
	}
	return false
}

func (u *USBHandler) getInterfaceAddress(name string) string {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return ""
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return ""
	}

	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && ipNet.IP.To4() != nil {
			return ipNet.IP.String()
		}
	}
	return ""
}
