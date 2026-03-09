//nolint:revive // "api" is a clear and appropriate package name
package api

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/fzdarsky/boardingpass/internal/logging"
)

// RegisterCaptivePortalRoutes adds handlers that suppress captive portal
// popups on iOS and Android when the device is connected to the BoardingPass
// WiFi AP (which has no internet access).
func RegisterCaptivePortalRoutes(mux *http.ServeMux) {
	// iOS captive portal detection
	mux.HandleFunc("/hotspot-detect.html", handleIOSCaptivePortal)

	// Android captive portal detection
	mux.HandleFunc("/generate_204", handleAndroidCaptivePortal)
}

// CaptivePortalServer runs a plain HTTP server on port 80 that responds to
// captive portal detection probes. iOS/Android send probes over HTTP (not
// HTTPS) to well-known domains. When combined with DNS redirection (e.g.,
// dnsmasq pointing captive.apple.com to the AP IP), this suppresses the
// "No Internet Connection" warning on phones.
type CaptivePortalServer struct {
	server *http.Server
	logger *logging.Logger
}

// NewCaptivePortalServer creates a captive portal HTTP server.
func NewCaptivePortalServer(logger *logging.Logger) *CaptivePortalServer {
	mux := http.NewServeMux()
	mux.HandleFunc("/hotspot-detect.html", handleIOSCaptivePortal)
	mux.HandleFunc("/generate_204", handleAndroidCaptivePortal)
	// Catch-all: iOS sometimes probes other paths
	mux.HandleFunc("/", handleIOSCaptivePortal)

	return &CaptivePortalServer{
		server: &http.Server{
			Handler:           mux,
			ReadTimeout:       5 * time.Second,
			WriteTimeout:      5 * time.Second,
			ReadHeaderTimeout: 5 * time.Second,
		},
		logger: logger,
	}
}

// Start begins listening on port 80 bound to the given address.
// Runs in a goroutine. Returns an error only if the listener cannot be created.
func (c *CaptivePortalServer) Start(address string) error {
	addr := address + ":80"
	lc := net.ListenConfig{}
	ln, err := lc.Listen(context.Background(), "tcp", addr)
	if err != nil {
		return err
	}

	c.logger.Info("captive portal HTTP server started", map[string]any{
		"address": addr,
	})

	go func() {
		if err := c.server.Serve(ln); err != nil && err != http.ErrServerClosed {
			c.logger.Warn("captive portal server error", map[string]any{
				"error": err.Error(),
			})
		}
	}()

	return nil
}

// Stop gracefully shuts down the captive portal server.
func (c *CaptivePortalServer) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	//nolint:errcheck // best-effort shutdown
	c.server.Shutdown(ctx)
}

func handleIOSCaptivePortal(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	//nolint:errcheck // best-effort response
	w.Write([]byte("<HTML><HEAD><TITLE>Success</TITLE></HEAD><BODY>Success</BODY></HTML>"))
}

func handleAndroidCaptivePortal(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}
