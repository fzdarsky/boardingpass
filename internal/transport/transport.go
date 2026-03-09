// Package transport provides lifecycle management for transient network transports.
package transport

import "context"

// Type identifies the kind of network transport.
type Type string

// Transport types.
const (
	TypeEthernet  Type = "ethernet"
	TypeWiFi      Type = "wifi"
	TypeBluetooth Type = "bluetooth"
	TypeUSB       Type = "usb"
)

// State represents the lifecycle state of a transport instance.
type State string

// Transport states.
const (
	StateDisabled State = "disabled"
	StateStarting State = "starting"
	StateActive   State = "active"
	StateFailed   State = "failed"
	StateStopping State = "stopping"
	StateStopped  State = "stopped"
)

// Transport represents a single configured transport instance.
type Transport struct {
	Type        Type
	Enabled     bool
	Interface   string
	Address     string
	Port        int
	State       State
	SystemdUnit string
}

// Handler defines the interface for transport lifecycle operations.
type Handler interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	TransportType() Type
	TransportState() State
}
