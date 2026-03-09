package mdns

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/fzdarsky/boardingpass/internal/logging"
)

const (
	mdnsAddr          = "224.0.0.251:5353"
	defaultTTL uint32 = 120 // seconds
)

// ServiceRecord holds the mDNS service information to announce.
type ServiceRecord struct {
	Instance string            // e.g. "BoardingPass-myhostname"
	Service  string            // "_boardingpass._tcp"
	Domain   string            // "local"
	Port     int               // 8443
	TXT      map[string]string // optional metadata
	Addrs    []net.IP          // IPv4 addresses to announce
}

// fqServiceName returns the fully qualified service name, e.g. "_boardingpass._tcp.local."
func (r *ServiceRecord) fqServiceName() string {
	return r.Service + "." + r.Domain + "."
}

// fqInstanceName returns the fully qualified instance name,
// e.g. "BoardingPass-host._boardingpass._tcp.local."
func (r *ServiceRecord) fqInstanceName() string {
	return r.Instance + "." + r.fqServiceName()
}

// fqHostName returns the fully qualified host name, e.g. "BoardingPass-host.local."
func (r *ServiceRecord) fqHostName() string {
	return r.Instance + "." + r.Domain + "."
}

// Announcer manages mDNS service announcements and query responses.
type Announcer struct {
	record ServiceRecord
	conn   *net.UDPConn
	logger *logging.Logger
	mu     sync.RWMutex
	cancel context.CancelFunc
	done   chan struct{}
}

// NewAnnouncer creates a new mDNS announcer for the given service record.
func NewAnnouncer(record ServiceRecord, logger *logging.Logger) *Announcer {
	return &Announcer{
		record: record,
		logger: logger,
		done:   make(chan struct{}),
	}
}

// Start begins mDNS announcements and query listening.
// The announcer joins the multicast group, sends initial announcements,
// and listens for queries in a background goroutine.
func (a *Announcer) Start(ctx context.Context) error {
	addr, err := net.ResolveUDPAddr("udp4", mdnsAddr)
	if err != nil {
		return fmt.Errorf("resolving mDNS address: %w", err)
	}

	conn, err := net.ListenMulticastUDP("udp4", nil, addr)
	if err != nil {
		return fmt.Errorf("joining mDNS multicast group: %w", err)
	}

	a.mu.Lock()
	a.conn = conn
	a.mu.Unlock()

	// Set read buffer size
	if err := conn.SetReadBuffer(65536); err != nil {
		a.logger.Warn("failed to set mDNS read buffer", map[string]any{
			"error": err.Error(),
		})
	}

	ctx, a.cancel = context.WithCancel(ctx)

	// Send initial announcements (RFC 6762 Section 8.3)
	a.announce()

	// Start query listener
	go a.listen(ctx)

	// Send follow-up announcements at t=1s and t=3s
	go func() {
		delays := []time.Duration{1 * time.Second, 2 * time.Second}
		for _, d := range delays {
			select {
			case <-ctx.Done():
				return
			case <-time.After(d):
				a.announce()
			}
		}
	}()

	a.logger.Info("mDNS announcer started", map[string]any{
		"instance": a.record.Instance,
		"service":  a.record.Service,
		"port":     a.record.Port,
		"addrs":    ipStrings(a.record.Addrs),
	})

	return nil
}

// Stop sends goodbye packets and shuts down the announcer.
func (a *Announcer) Stop() {
	if a.cancel != nil {
		a.cancel()
	}

	// Send goodbye (TTL=0)
	a.sendRecords(0)

	a.mu.Lock()
	conn := a.conn
	a.conn = nil
	a.mu.Unlock()

	if conn != nil {
		_ = conn.Close()
	}

	// Wait for listener to exit
	<-a.done

	a.logger.Info("mDNS announcer stopped")
}

// AddAddress adds an IP address and re-announces the service.
func (a *Announcer) AddAddress(ip net.IP) {
	v4 := ip.To4()
	if v4 == nil {
		return
	}

	a.mu.Lock()
	// Check for duplicate
	for _, existing := range a.record.Addrs {
		if existing.Equal(v4) {
			a.mu.Unlock()
			return
		}
	}
	a.record.Addrs = append(a.record.Addrs, v4)
	a.mu.Unlock()

	a.logger.Info("mDNS address added", map[string]any{
		"address": ip.String(),
	})
	a.announce()
}

// RemoveAddress removes an IP address and re-announces the service.
func (a *Announcer) RemoveAddress(ip net.IP) {
	v4 := ip.To4()
	if v4 == nil {
		return
	}

	a.mu.Lock()
	for i, existing := range a.record.Addrs {
		if existing.Equal(v4) {
			a.record.Addrs = append(a.record.Addrs[:i], a.record.Addrs[i+1:]...)
			break
		}
	}
	a.mu.Unlock()

	a.logger.Info("mDNS address removed", map[string]any{
		"address": ip.String(),
	})
	a.announce()
}

func (a *Announcer) announce() {
	a.sendRecords(defaultTTL)
}

func (a *Announcer) sendRecords(ttl uint32) {
	msg := a.buildResponse(ttl)

	data, err := PackMessage(msg)
	if err != nil {
		a.logger.Warn("failed to pack mDNS announcement", map[string]any{
			"error": err.Error(),
		})
		return
	}

	a.mu.RLock()
	conn := a.conn
	a.mu.RUnlock()

	if conn == nil {
		return
	}

	dst, _ := net.ResolveUDPAddr("udp4", mdnsAddr)
	if _, err := conn.WriteTo(data, dst); err != nil {
		a.logger.Warn("failed to send mDNS announcement", map[string]any{
			"error": err.Error(),
		})
	}
}

func (a *Announcer) listen(ctx context.Context) {
	defer close(a.done)

	buf := make([]byte, 65536)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		a.mu.RLock()
		conn := a.conn
		a.mu.RUnlock()
		if conn == nil {
			return
		}

		// Set read deadline so we can check context cancellation
		if err := conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond)); err != nil {
			return
		}

		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue // read deadline expired, check context
			}
			// Connection closed or other error
			return
		}

		a.handleQuery(buf[:n])
	}
}

func (a *Announcer) handleQuery(data []byte) {
	msg, err := UnpackMessage(data)
	if err != nil {
		return // silently ignore malformed messages
	}

	// Only respond to queries (QR=0)
	if msg.Header.Flags&flagQR != 0 {
		return
	}

	// Check if any question matches our service
	for _, q := range msg.Questions {
		qName := strings.ToLower(q.Name)
		if a.matchesQuestion(qName, q.Type) {
			a.announce()
			return // one response per query is sufficient
		}
	}
}

func (a *Announcer) matchesQuestion(qName string, qType uint16) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	svcName := strings.ToLower(a.record.fqServiceName())
	instName := strings.ToLower(a.record.fqInstanceName())
	hostName := strings.ToLower(a.record.fqHostName())

	switch {
	case qType == TypePTR && qName == svcName:
		return true
	case qType == TypeSRV && qName == instName:
		return true
	case qType == TypeTXT && qName == instName:
		return true
	case qType == TypeA && qName == hostName:
		return true
	// Also respond to DNS-SD browse queries
	case qType == TypePTR && qName == "_services._dns-sd._udp."+strings.ToLower(a.record.Domain)+".":
		return true
	default:
		return false
	}
}

// buildResponse constructs a full mDNS response with PTR, SRV, TXT, and A records.
func (a *Announcer) buildResponse(ttl uint32) *Message {
	a.mu.RLock()
	defer a.mu.RUnlock()

	instName := a.record.fqInstanceName()
	svcName := a.record.fqServiceName()
	hostName := a.record.fqHostName()

	ptrData, _ := NewPTRRecord(instName)
	srvData, _ := NewSRVRecord(0, 0, uint16(a.record.Port), hostName) //nolint:gosec // port is validated 1-65535
	txtData := NewTXTRecord(a.record.TXT)

	msg := &Message{
		Header: Header{
			Flags: flagQR | flagAA, // Response, Authoritative
		},
		Answers: []ResourceRecord{
			{
				Name:  svcName,
				Type:  TypePTR,
				Class: ClassIN, // PTR records don't use cache-flush
				TTL:   ttl,
				Data:  ptrData,
			},
		},
		Additional: []ResourceRecord{
			{
				Name:  instName,
				Type:  TypeSRV,
				Class: ClassINFlush,
				TTL:   ttl,
				Data:  srvData,
			},
			{
				Name:  instName,
				Type:  TypeTXT,
				Class: ClassINFlush,
				TTL:   ttl,
				Data:  txtData,
			},
		},
	}

	// Add A record for each address
	for _, ip := range a.record.Addrs {
		aData := NewARecord(ip)
		if aData == nil {
			continue
		}
		msg.Additional = append(msg.Additional, ResourceRecord{
			Name:  hostName,
			Type:  TypeA,
			Class: ClassINFlush,
			TTL:   ttl,
			Data:  aData,
		})
	}

	return msg
}

func ipStrings(ips []net.IP) []string {
	s := make([]string, len(ips))
	for i, ip := range ips {
		s[i] = ip.String()
	}
	return s
}
