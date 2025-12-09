// Package api provides the HTTP/HTTPS server and handlers for the BoardingPass API.
//
//nolint:revive // "api" is a clear and appropriate package name
package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/fzdarsky/boardingpass/internal/config"
	"github.com/fzdarsky/boardingpass/internal/logging"
	tlspkg "github.com/fzdarsky/boardingpass/internal/tls"
)

// Server represents the HTTP/HTTPS API server.
type Server struct {
	httpServer *http.Server
	logger     *logging.Logger
	config     *config.Config
}

// New creates a new API server instance.
func New(cfg *config.Config, logger *logging.Logger) (*Server, error) {
	mux := http.NewServeMux()

	// Routes will be registered here by the main package
	// This is a placeholder for the server structure
	// Actual route registration happens in cmd/boardingpass/main.go

	server := &Server{
		httpServer: &http.Server{
			Addr:              fmt.Sprintf("%s:%d", cfg.Transports.Ethernet.Address, cfg.Transports.Ethernet.Port),
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
	if cfg.Transports.Ethernet.Enabled {
		tlsConfig, err := tlspkg.NewServerConfig(
			cfg.Transports.Ethernet.TLSCert,
			cfg.Transports.Ethernet.TLSKey,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create TLS config: %w", err)
		}
		server.httpServer.TLSConfig = tlsConfig
	}

	return server, nil
}

// Start begins serving HTTPS requests.
func (s *Server) Start(ctx context.Context) error {
	if !s.config.Transports.Ethernet.Enabled {
		return fmt.Errorf("no transports enabled")
	}

	s.logger.Info("starting HTTPS server", map[string]any{
		"address": s.httpServer.Addr,
	})

	// Start server in a goroutine
	errChan := make(chan error, 1)
	go func() {
		// ListenAndServeTLS starts the HTTPS server
		// We pass empty strings because TLS config is already set in httpServer.TLSConfig
		if err := s.httpServer.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// Wait for context cancellation or error
	select {
	case err := <-errChan:
		return fmt.Errorf("server error: %w", err)
	case <-ctx.Done():
		s.logger.Info("shutting down HTTPS server")
		return s.Shutdown(context.Background())
	}
}

// Shutdown gracefully shuts down the server.
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
