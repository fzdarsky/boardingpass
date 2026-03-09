package transport

import (
	"context"
	"sync"

	"github.com/fzdarsky/boardingpass/internal/config"
	"github.com/fzdarsky/boardingpass/internal/logging"
)

// Manager orchestrates the lifecycle of all configured transports.
type Manager struct {
	cfg      *config.Config
	logger   *logging.Logger
	handlers []Handler
	mu       sync.Mutex
}

// NewManager creates a new transport manager.
func NewManager(cfg *config.Config, logger *logging.Logger) *Manager {
	return &Manager{
		cfg:    cfg,
		logger: logger,
	}
}

// Register adds a transport handler to the manager.
func (m *Manager) Register(h Handler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers = append(m.handlers, h)
}

// StartAll starts all registered transports. Failures are non-fatal:
// a warning is logged and remaining transports continue starting.
func (m *Manager) StartAll(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, h := range m.handlers {
		m.logger.Info("starting transport", map[string]any{
			"transport": string(h.TransportType()),
		})

		if err := h.Start(ctx); err != nil {
			m.logger.Warn("transport failed to start, continuing with remaining transports", map[string]any{
				"transport": string(h.TransportType()),
				"error":     err.Error(),
			})
			continue
		}

		m.logger.Info("transport started", map[string]any{
			"transport": string(h.TransportType()),
		})
	}

	return nil
}

// StopAll stops all active transports in reverse order.
func (m *Manager) StopAll(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Stop in reverse order
	for i := len(m.handlers) - 1; i >= 0; i-- {
		h := m.handlers[i]
		if h.TransportState() != StateActive {
			continue
		}

		m.logger.Info("stopping transport", map[string]any{
			"transport": string(h.TransportType()),
		})

		if err := h.Stop(ctx); err != nil {
			m.logger.Warn("transport failed to stop cleanly", map[string]any{
				"transport": string(h.TransportType()),
				"error":     err.Error(),
			})
		}
	}

	return nil
}

// ActiveTransports returns all transports currently in active state.
func (m *Manager) ActiveTransports() []Transport {
	m.mu.Lock()
	defer m.mu.Unlock()

	var active []Transport
	for _, h := range m.handlers {
		if h.TransportState() == StateActive {
			active = append(active, Transport{
				Type:  h.TransportType(),
				State: StateActive,
			})
		}
	}
	return active
}
