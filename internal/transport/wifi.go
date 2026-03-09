package transport

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fzdarsky/boardingpass/internal/config"
	"github.com/fzdarsky/boardingpass/internal/logging"
)

const (
	// runtimeDir is where generated configs are written (created by systemd RuntimeDirectory).
	runtimeDir = "/run/boardingpass"
)

// WiFiHandler manages the WiFi AP transport lifecycle via systemd.
type WiFiHandler struct {
	cfg           config.WiFiTransport
	logger        *logging.Logger
	state         State
	resolvedIface string // set during Start, used by Stop
	mu            sync.Mutex
}

// NewWiFiHandler creates a new WiFi transport handler.
func NewWiFiHandler(cfg config.WiFiTransport, logger *logging.Logger) *WiFiHandler {
	return &WiFiHandler{
		cfg:    cfg,
		logger: logger,
		state:  StateDisabled,
	}
}

// Start activates the WiFi AP by generating configs and starting systemd units.
func (w *WiFiHandler) Start(ctx context.Context) error {
	w.mu.Lock()
	w.state = StateStarting
	w.mu.Unlock()

	// Check dependencies
	for _, bin := range []string{"hostapd", "dnsmasq"} {
		if _, err := exec.LookPath(bin); err != nil {
			w.setState(StateFailed)
			return fmt.Errorf("%s binary not found: %w", bin, err)
		}
	}

	// Resolve interface: use config value or auto-detect
	iface, err := w.resolveInterface()
	if err != nil {
		w.setState(StateFailed)
		return err
	}
	w.mu.Lock()
	w.resolvedIface = iface
	w.mu.Unlock()

	// Validate interface exists
	if _, err := os.Stat(filepath.Join("/sys/class/net", iface)); err != nil {
		w.setState(StateFailed)
		return fmt.Errorf("wifi interface %s not found: %w", iface, err)
	}

	// Resolve SSID
	ssid := w.cfg.SSID
	if ssid == "" {
		hostname, _ := os.Hostname()
		if hostname == "" {
			hostname = "device"
		}
		ssid = "BoardingPass-" + hostname
	}

	// Resolve channel
	channel := w.cfg.Channel
	if channel == 0 {
		channel = 6
	}

	// Resolve address
	address := w.cfg.Address
	if address == "" {
		address = "10.0.0.1"
	}

	w.logger.Info("wifi transport resolved configuration", map[string]any{
		"interface": iface,
		"ssid":      ssid,
		"channel":   channel,
		"address":   address,
	})

	// Generate config files
	if err := w.generateHostapdConf(iface, ssid, channel); err != nil {
		w.setState(StateFailed)
		return fmt.Errorf("failed to generate hostapd config: %w", err)
	}

	if err := w.generateDnsmasqConf(iface, address); err != nil {
		w.setState(StateFailed)
		return fmt.Errorf("failed to generate dnsmasq config: %w", err)
	}

	if err := w.generateEnvFile(iface, address); err != nil {
		w.setState(StateFailed)
		return fmt.Errorf("failed to generate environment file: %w", err)
	}

	// Start WiFi AP systemd unit
	wifiUnit := fmt.Sprintf("boardingpass-wifi@%s", iface)
	//nolint:gosec // G204: interface name is validated above
	cmd := exec.CommandContext(ctx, "sudo", "systemctl", "start", wifiUnit)
	if out, err := cmd.CombinedOutput(); err != nil {
		w.setState(StateFailed)
		return fmt.Errorf("failed to start %s: %s: %w", wifiUnit, string(out), err)
	}

	// Start dnsmasq companion unit (non-fatal — WiFi AP works without DHCP)
	dnsmasqUnit := fmt.Sprintf("boardingpass-dnsmasq@%s", iface)
	//nolint:gosec // G204: interface name is validated above
	cmd = exec.CommandContext(ctx, "sudo", "systemctl", "start", dnsmasqUnit)
	if out, err := cmd.CombinedOutput(); err != nil {
		w.logger.Warn("dnsmasq failed to start (WiFi AP still active, but no DHCP)", map[string]any{
			"unit":  dnsmasqUnit,
			"error": err.Error(),
			"out":   string(out),
		})
	}

	w.setState(StateActive)
	return nil
}

// Stop deactivates the WiFi AP by stopping systemd units.
func (w *WiFiHandler) Stop(ctx context.Context) error {
	w.mu.Lock()
	w.state = StateStopping
	iface := w.resolvedIface
	w.mu.Unlock()

	if iface == "" {
		iface = w.cfg.Interface
	}

	// Stop dnsmasq first (non-fatal)
	dnsmasqUnit := fmt.Sprintf("boardingpass-dnsmasq@%s", iface)
	//nolint:gosec // G204: interface name from validated config
	cmd := exec.CommandContext(ctx, "sudo", "systemctl", "stop", dnsmasqUnit)
	if out, err := cmd.CombinedOutput(); err != nil {
		w.logger.Warn("failed to stop dnsmasq", map[string]any{
			"unit":  dnsmasqUnit,
			"error": err.Error(),
			"out":   string(out),
		})
	}

	// Stop WiFi AP
	wifiUnit := fmt.Sprintf("boardingpass-wifi@%s", iface)
	//nolint:gosec // G204: interface name from validated config
	cmd = exec.CommandContext(ctx, "sudo", "systemctl", "stop", wifiUnit)
	if out, err := cmd.CombinedOutput(); err != nil {
		w.setState(StateFailed)
		return fmt.Errorf("failed to stop %s: %s: %w", wifiUnit, string(out), err)
	}

	w.setState(StateStopped)
	return nil
}

// TransportType returns the transport type.
func (w *WiFiHandler) TransportType() Type {
	return TypeWiFi
}

// TransportState returns the current state.
func (w *WiFiHandler) TransportState() State {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.state
}

func (w *WiFiHandler) setState(s State) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.state = s
}

// resolveInterface returns the WiFi interface to use: from config or auto-detected.
func (w *WiFiHandler) resolveInterface() (string, error) {
	if w.cfg.Interface != "" {
		return w.cfg.Interface, nil
	}
	return detectWiFiInterface()
}

// detectWiFiInterface scans /sys/class/net/ for wireless interfaces.
// A wireless interface is identified by the presence of a "wireless" subdirectory
// (created by the cfg80211 kernel subsystem).
func detectWiFiInterface() (string, error) {
	entries, err := os.ReadDir("/sys/class/net")
	if err != nil {
		return "", fmt.Errorf("failed to read /sys/class/net: %w", err)
	}

	for _, entry := range entries {
		name := entry.Name()
		wirelessPath := filepath.Join("/sys/class/net", name, "wireless")
		if _, err := os.Stat(wirelessPath); err == nil {
			return name, nil
		}
	}

	return "", fmt.Errorf("no wireless interface found")
}

// generateHostapdConf writes a hostapd configuration file for the given interface.
func (w *WiFiHandler) generateHostapdConf(iface, ssid string, channel int) error {
	var b strings.Builder

	fmt.Fprintf(&b, "interface=%s\n", iface)
	b.WriteString("driver=nl80211\n")
	fmt.Fprintf(&b, "ssid=%s\n", ssid)
	b.WriteString("hw_mode=g\n")
	fmt.Fprintf(&b, "channel=%d\n", channel)
	b.WriteString("ieee80211n=1\n")
	b.WriteString("wmm_enabled=1\n")
	b.WriteString("macaddr_acl=0\n")
	b.WriteString("auth_algs=1\n")

	if w.cfg.Password != "" {
		b.WriteString("wpa=2\n")
		fmt.Fprintf(&b, "wpa_passphrase=%s\n", w.cfg.Password)
		b.WriteString("wpa_key_mgmt=WPA-PSK\n")
		b.WriteString("rsn_pairwise=CCMP\n")
	}

	path := filepath.Join(runtimeDir, fmt.Sprintf("hostapd-%s.conf", iface))
	//nolint:gosec // G306: config file, not secrets (password handled by hostapd)
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

// generateDnsmasqConf writes a dnsmasq configuration file for DHCP and DNS on the AP.
func (w *WiFiHandler) generateDnsmasqConf(iface, address string) error {
	// Derive DHCP range from the gateway address (e.g., 10.0.0.1 → 10.0.0.10-10.0.0.50)
	rangeStart, rangeEnd := dhcpRange(address)

	var b strings.Builder
	fmt.Fprintf(&b, "interface=%s\n", iface)
	b.WriteString("bind-interfaces\n")
	fmt.Fprintf(&b, "listen-address=%s\n", address)
	fmt.Fprintf(&b, "dhcp-range=%s,%s,255.255.255.0,12h\n", rangeStart, rangeEnd)
	// Wildcard DNS redirect — all DNS queries resolve to the AP IP.
	// This enables captive portal detection by iOS/Android.
	fmt.Fprintf(&b, "address=/#/%s\n", address)
	b.WriteString("no-resolv\n")
	b.WriteString("no-hosts\n")

	path := filepath.Join(runtimeDir, fmt.Sprintf("dnsmasq-%s.conf", iface))
	//nolint:gosec // G306: config file, not secrets
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

// generateEnvFile writes a systemd environment file with the AP address for the WiFi unit.
func (w *WiFiHandler) generateEnvFile(iface, address string) error {
	content := fmt.Sprintf("BOARDINGPASS_WIFI_ADDRESS=%s\nBOARDINGPASS_WIFI_NETMASK=24\n", address)
	path := filepath.Join(runtimeDir, fmt.Sprintf("wifi-%s.env", iface))
	//nolint:gosec // G306: non-sensitive environment variables
	return os.WriteFile(path, []byte(content), 0o644)
}

// dhcpRange derives DHCP start/end addresses from a gateway IP.
// For "10.0.0.1" → ("10.0.0.10", "10.0.0.50").
func dhcpRange(gateway string) (string, string) {
	ip := net.ParseIP(gateway)
	if ip == nil {
		// Fallback to safe defaults
		return "10.0.0.10", "10.0.0.50"
	}

	ip4 := ip.To4()
	if ip4 == nil {
		return "10.0.0.10", "10.0.0.50"
	}

	startIP := make(net.IP, 4)
	copy(startIP, ip4)
	startIP[3] = 10

	endIP := make(net.IP, 4)
	copy(endIP, ip4)
	endIP[3] = 50

	return startIP.String(), endIP.String()
}
