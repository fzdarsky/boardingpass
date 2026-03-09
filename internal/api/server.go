// Package api provides the HTTP/HTTPS server and handlers for the BoardingPass API.
//
//nolint:revive // "api" is a clear and appropriate package name
package api

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/fzdarsky/boardingpass/internal/config"
	"github.com/fzdarsky/boardingpass/internal/logging"
	tlspkg "github.com/fzdarsky/boardingpass/internal/tls"
)

// Server represents the HTTP/HTTPS API server with multi-listener support.
type Server struct {
	httpServer *http.Server
	tlsConfig  *tls.Config
	listeners  []net.Listener
	logger     *logging.Logger
	config     *config.Config
	mu         sync.Mutex
}

// New creates a new API server instance.
func New(cfg *config.Config, logger *logging.Logger) (*Server, error) {
	mux := http.NewServeMux()

	server := &Server{
		httpServer: &http.Server{
			Handler:           mux,
			ReadTimeout:       30 * time.Second,
			WriteTimeout:      30 * time.Second,
			IdleTimeout:       120 * time.Second,
			ReadHeaderTimeout: 10 * time.Second,
		},
		logger: logger,
		config: cfg,
	}

	// Configure TLS
	tlsCfg, err := tlspkg.NewServerConfig(
		cfg.Service.TLSCert,
		cfg.Service.TLSKey,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create TLS config: %w", err)
	}
	server.httpServer.TLSConfig = tlsCfg
	server.tlsConfig = tlsCfg

	return server, nil
}

// Start begins serving HTTPS requests on all configured listeners.
func (s *Server) Start(ctx context.Context) error {
	addrs := s.configuredAddresses()
	if len(addrs) == 0 {
		return fmt.Errorf("no transports enabled")
	}

	errChan := make(chan error, len(addrs))

	for _, addr := range addrs {
		ln, err := s.createListener(addr)
		if err != nil {
			s.logger.Warn("failed to create listener", map[string]any{
				"address": addr,
				"error":   err.Error(),
			})
			continue
		}

		s.mu.Lock()
		s.listeners = append(s.listeners, ln)
		s.mu.Unlock()

		s.logger.Info("HTTPS listener started", map[string]any{
			"address": addr,
		})

		go func(l net.Listener) {
			if err := s.httpServer.Serve(l); err != nil && err != http.ErrServerClosed {
				errChan <- err
			}
		}(ln)
	}

	if len(s.listeners) == 0 {
		return fmt.Errorf("no listeners could be created")
	}

	// Wait for context cancellation or error
	select {
	case err := <-errChan:
		return fmt.Errorf("server error: %w", err)
	case <-ctx.Done():
		s.logger.Info("shutting down HTTPS server")
		return s.Shutdown(context.Background())
	}
}

// AddListener creates and starts serving on a new address at runtime.
func (s *Server) AddListener(address string, port int) error {
	addr := fmt.Sprintf("%s:%d", address, port)

	ln, err := s.createListener(addr)
	if err != nil {
		return fmt.Errorf("failed to create listener on %s: %w", addr, err)
	}

	s.mu.Lock()
	s.listeners = append(s.listeners, ln)
	s.mu.Unlock()

	s.logger.Info("added HTTPS listener", map[string]any{
		"address": addr,
	})

	go func() {
		if err := s.httpServer.Serve(ln); err != nil && err != http.ErrServerClosed {
			s.logger.Warn("listener error", map[string]any{
				"address": addr,
				"error":   err.Error(),
			})
		}
	}()

	return nil
}

// RemoveListener closes the listener bound to the given address.
func (s *Server) RemoveListener(address string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, ln := range s.listeners {
		if ln.Addr().String() == address {
			s.logger.Info("removing HTTPS listener", map[string]any{
				"address": address,
			})
			if err := ln.Close(); err != nil {
				return fmt.Errorf("failed to close listener %s: %w", address, err)
			}
			s.listeners = append(s.listeners[:i], s.listeners[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("no listener found for address %s", address)
}

// Shutdown gracefully shuts down the server and all listeners.
func (s *Server) Shutdown(ctx context.Context) error {
	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	s.logger.Info("server shutdown complete")
	return nil
}

// Handler returns the HTTP handler for route registration.
func (s *Server) Handler() *http.ServeMux {
	if mux, ok := s.httpServer.Handler.(*http.ServeMux); ok {
		return mux
	}
	return nil
}

// RegisterRoute registers a handler for a specific HTTP path.
func (s *Server) RegisterRoute(pattern string, handler http.Handler) {
	if mux := s.Handler(); mux != nil {
		mux.Handle(pattern, handler)
	}
}

// RegisterRouteFunc registers a handler function for a specific HTTP path.
func (s *Server) RegisterRouteFunc(pattern string, handler http.HandlerFunc) {
	if mux := s.Handler(); mux != nil {
		mux.HandleFunc(pattern, handler)
	}
}

// configuredAddresses returns all addresses to listen on based on config.
func (s *Server) configuredAddresses() []string {
	var addrs []string

	port := s.config.Service.Port

	if s.config.Transports.Ethernet.Enabled {
		addrs = append(addrs, fmt.Sprintf("%s:%d",
			s.config.Transports.Ethernet.Address, port,
		))
	}

	if s.config.Transports.WiFi.Enabled {
		addrs = append(addrs, fmt.Sprintf("%s:%d",
			s.config.Transports.WiFi.Address, port,
		))
	}

	if s.config.Transports.Bluetooth.Enabled {
		addrs = append(addrs, fmt.Sprintf("%s:%d",
			s.config.Transports.Bluetooth.Address, port,
		))
	}

	// USB addresses are added dynamically via AddListener

	return addrs
}

// createListener creates a TLS listener on the given address.
func (s *Server) createListener(addr string) (net.Listener, error) {
	lc := net.ListenConfig{}
	ln, err := lc.Listen(context.Background(), "tcp", addr)
	if err != nil {
		return nil, err
	}

	if s.tlsConfig != nil {
		ln = tls.NewListener(ln, s.tlsConfig)
	}

	return ln, nil
}
