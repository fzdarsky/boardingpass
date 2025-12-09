// BoardingPass is a lightweight, ephemeral bootstrap service for headless Linux devices.
package main

import (
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fzdarsky/boardingpass/internal/api"
	"github.com/fzdarsky/boardingpass/internal/api/handlers"
	"github.com/fzdarsky/boardingpass/internal/api/middleware"
	"github.com/fzdarsky/boardingpass/internal/auth"
	"github.com/fzdarsky/boardingpass/internal/config"
	"github.com/fzdarsky/boardingpass/internal/logging"
)

var (
	// version is set by build flags
	version = "dev"
	// commit is set by build flags
	commit = "none"
)

func main() {
	// Parse command-line flags
	configPath := flag.String("config", "/etc/boardingpass/config.yaml", "path to configuration file")
	verifierPath := flag.String("verifier", "/etc/boardingpass/verifier", "path to SRP verifier file")
	flag.Parse()

	// Initialize logger with default settings (will be updated from config)
	logger := logging.New(logging.LevelInfo, logging.FormatJSON)

	// Run the service
	if err := run(*configPath, *verifierPath, logger); err != nil {
		logger.Error("service failed", map[string]any{
			"error": err.Error(),
		})
		os.Exit(1)
	}
}

func run(configPath, verifierPath string, logger *logging.Logger) error {
	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Parse log level and format from config
	logLevel := parseLogLevel(cfg.Logging.Level)
	logFormat := parseLogFormat(cfg.Logging.Format)

	// Recreate logger with config settings
	logger = logging.New(logLevel, logFormat)

	// Log startup information
	logger.Info("BoardingPass service starting", map[string]any{
		"version":        version,
		"commit":         commit,
		"log_level":      cfg.Logging.Level,
		"log_format":     cfg.Logging.Format,
		"listen_address": fmt.Sprintf("%s:%d", cfg.Transports.Ethernet.Address, cfg.Transports.Ethernet.Port),
		"session_ttl":    cfg.Service.SessionTTL,
		"sentinel_file":  cfg.Service.SentinelFile,
		"commands_count": len(cfg.Commands),
	})

	// Check sentinel file - if it exists, the device is already provisioned
	if _, err := os.Stat(cfg.Service.SentinelFile); err == nil {
		logger.Info("sentinel file exists - device already provisioned, exiting", map[string]any{
			"sentinel_file": cfg.Service.SentinelFile,
		})
		return nil
	}

	// Load SRP verifier configuration
	verifierCfg, err := auth.LoadVerifierConfig(verifierPath)
	if err != nil {
		return fmt.Errorf("failed to load verifier config: %w", err)
	}

	// Parse session TTL
	sessionTTL, err := time.ParseDuration(cfg.Service.SessionTTL)
	if err != nil {
		return fmt.Errorf("failed to parse session TTL: %w", err)
	}

	// Generate a random HMAC secret for session tokens
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return fmt.Errorf("failed to generate session secret: %w", err)
	}

	// Initialize session manager
	sessionManager := auth.NewSessionManager(secret, sessionTTL)

	// Initialize rate limiter
	rateLimiter := auth.NewRateLimiter()

	// Initialize SRP store with 5-minute TTL for SRP sessions
	srpStore := auth.NewSRPStore(5 * time.Minute)

	// Create a standard library logger for auth handler
	stdLogger := log.New(os.Stdout, "", log.LstdFlags)

	// Create auth handler
	authHandler := handlers.NewAuthHandler(verifierCfg, sessionManager, rateLimiter, srpStore, stdLogger)

	// Create API server
	server, err := api.New(cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	// Get the HTTP mux for route registration
	mux := server.Handler()
	if mux == nil {
		return fmt.Errorf("failed to get server handler")
	}

	// Create authentication middleware
	authMiddleware := middleware.NewAuthMiddleware(sessionManager)

	// Register routes
	// Auth endpoints (no authentication required)
	mux.HandleFunc("/auth/srp/init", authHandler.HandleSRPInit)
	mux.HandleFunc("/auth/srp/verify", authHandler.HandleSRPVerify)

	// Info endpoint (requires authentication)
	infoHandler := handlers.NewInfoHandler()
	mux.Handle("/info", authMiddleware.Require(infoHandler))

	// Network endpoint (requires authentication)
	networkHandler := handlers.NewNetworkHandler()
	mux.Handle("/network", authMiddleware.Require(networkHandler))

	// TODO: Add /configure endpoint (US3 - not yet implemented)
	// TODO: Add /command endpoint (US4 - not yet implemented)
	// TODO: Add /complete endpoint (US5 - not yet implemented)

	// Set up signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a channel to listen for interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	// Handle signals in a goroutine
	go func() {
		sig := <-sigChan
		logger.Info("received shutdown signal", map[string]any{
			"signal": sig.String(),
		})
		cancel()
	}()

	// Start the server
	logger.Info("HTTPS server ready to accept connections")

	// Notify systemd that the service is ready (Type=notify)
	notifySystemd("READY=1")

	if err := server.Start(ctx); err != nil {
		return fmt.Errorf("server failed: %w", err)
	}

	logger.Info("BoardingPass service stopped")

	// Notify systemd that the service is stopping
	notifySystemd("STOPPING=1")

	return nil
}

// notifySystemd sends a notification to systemd if NOTIFY_SOCKET is set.
// This enables systemd Type=notify service management.
func notifySystemd(state string) {
	notifySocket := os.Getenv("NOTIFY_SOCKET")
	if notifySocket == "" {
		// Not running under systemd with Type=notify
		return
	}

	// Simple implementation: write to the socket
	// For production, consider using github.com/coreos/go-systemd/daemon.SdNotify
	// However, per project constitution (minimal dependencies), we'll use a simple approach
	conn, err := net.DialUnix("unixgram", nil, &net.UnixAddr{Name: notifySocket, Net: "unixgram"})
	if err != nil {
		// Silently ignore errors - systemd notification is optional
		return
	}
	defer func() {
		_ = conn.Close()
	}()

	_, _ = conn.Write([]byte(state))
}

func parseLogLevel(level string) logging.LogLevel {
	switch level {
	case "debug":
		return logging.LevelDebug
	case "info":
		return logging.LevelInfo
	case "warn":
		return logging.LevelWarn
	case "error":
		return logging.LevelError
	default:
		return logging.LevelInfo
	}
}

func parseLogFormat(format string) logging.LogFormat {
	switch format {
	case "json":
		return logging.FormatJSON
	case "human":
		return logging.FormatHuman
	default:
		return logging.FormatJSON
	}
}
