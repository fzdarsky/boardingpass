// Package mdns implements a minimal mDNS/DNS-SD responder for service announcement.
//
// Only the DNS record types needed for DNS-SD are supported: A, PTR, SRV, and TXT.
// The wire format follows RFC 1035 (DNS) and RFC 6762 (mDNS).
package mdns

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"strings"
)

// DNS record type constants (RFC 1035, RFC 2782).
const (
	TypeA   uint16 = 1
	TypePTR uint16 = 12
	TypeTXT uint16 = 16
	TypeSRV uint16 = 33
)

// DNS class constants.
const (
	ClassIN        uint16 = 1
	ClassINFlush   uint16 = 1 | 0x8000 // Cache-flush bit set (RFC 6762 Section 10.2)
	ClassINUnicast uint16 = 1 | 0x8000 // QU bit in questions (RFC 6762 Section 5.4)
)

// DNS header flag bits.
const (
	flagQR uint16 = 1 << 15 // Query/Response
	flagAA uint16 = 1 << 10 // Authoritative Answer
)

const headerSize = 12

// Header represents a DNS message header (RFC 1035 Section 4.1.1).
type Header struct {
	ID      uint16
	Flags   uint16
	QDCount uint16
	ANCount uint16
	NSCount uint16
	ARCount uint16
}

// Question represents a DNS question entry (RFC 1035 Section 4.1.2).
type Question struct {
	Name  string
	Type  uint16
	Class uint16
}

// ResourceRecord represents a DNS resource record (RFC 1035 Section 4.1.3).
type ResourceRecord struct {
	Name  string
	Type  uint16
	Class uint16
	TTL   uint32
	Data  []byte
}

// Message represents a complete DNS message.
type Message struct {
	Header     Header
	Questions  []Question
	Answers    []ResourceRecord
	Authority  []ResourceRecord
	Additional []ResourceRecord
}

// PackMessage serializes a DNS message to wire format.
func PackMessage(msg *Message) ([]byte, error) {
	buf := make([]byte, 0, 512) // typical mDNS packet size

	// Header
	h := make([]byte, headerSize)
	binary.BigEndian.PutUint16(h[0:2], msg.Header.ID)
	binary.BigEndian.PutUint16(h[2:4], msg.Header.Flags)
	binary.BigEndian.PutUint16(h[4:6], uint16(len(msg.Questions)))    //nolint:gosec // bounded by DNS protocol
	binary.BigEndian.PutUint16(h[6:8], uint16(len(msg.Answers)))      //nolint:gosec // bounded by DNS protocol
	binary.BigEndian.PutUint16(h[8:10], uint16(len(msg.Authority)))   //nolint:gosec // bounded by DNS protocol
	binary.BigEndian.PutUint16(h[10:12], uint16(len(msg.Additional))) //nolint:gosec // bounded by DNS protocol
	buf = append(buf, h...)

	// Questions
	for _, q := range msg.Questions {
		name, err := encodeName(q.Name)
		if err != nil {
			return nil, fmt.Errorf("encoding question name %q: %w", q.Name, err)
		}
		buf = append(buf, name...)
		buf = binary.BigEndian.AppendUint16(buf, q.Type)
		buf = binary.BigEndian.AppendUint16(buf, q.Class)
	}

	// Resource record sections
	for _, rr := range msg.Answers {
		b, err := packRR(&rr)
		if err != nil {
			return nil, err
		}
		buf = append(buf, b...)
	}
	for _, rr := range msg.Authority {
		b, err := packRR(&rr)
		if err != nil {
			return nil, err
		}
		buf = append(buf, b...)
	}
	for _, rr := range msg.Additional {
		b, err := packRR(&rr)
		if err != nil {
			return nil, err
		}
		buf = append(buf, b...)
	}

	return buf, nil
}

func packRR(rr *ResourceRecord) ([]byte, error) {
	name, err := encodeName(rr.Name)
	if err != nil {
		return nil, fmt.Errorf("encoding RR name %q: %w", rr.Name, err)
	}

	buf := make([]byte, 0, len(name)+10+len(rr.Data))
	buf = append(buf, name...)
	buf = binary.BigEndian.AppendUint16(buf, rr.Type)
	buf = binary.BigEndian.AppendUint16(buf, rr.Class)
	buf = binary.BigEndian.AppendUint32(buf, rr.TTL)
	buf = binary.BigEndian.AppendUint16(buf, uint16(len(rr.Data))) //nolint:gosec // RDATA bounded by DNS packet size
	buf = append(buf, rr.Data...)

	return buf, nil
}

// UnpackMessage deserializes a DNS message from wire format.
func UnpackMessage(data []byte) (*Message, error) {
	if len(data) < headerSize {
		return nil, errors.New("message too short for header")
	}

	msg := &Message{}
	msg.Header = Header{
		ID:      binary.BigEndian.Uint16(data[0:2]),
		Flags:   binary.BigEndian.Uint16(data[2:4]),
		QDCount: binary.BigEndian.Uint16(data[4:6]),
		ANCount: binary.BigEndian.Uint16(data[6:8]),
		NSCount: binary.BigEndian.Uint16(data[8:10]),
		ARCount: binary.BigEndian.Uint16(data[10:12]),
	}

	off := headerSize

	// Questions
	for range msg.Header.QDCount {
		name, newOff, err := decodeName(data, off)
		if err != nil {
			return nil, fmt.Errorf("decoding question name: %w", err)
		}
		off = newOff
		if off+4 > len(data) {
			return nil, errors.New("question truncated")
		}
		q := Question{
			Name:  name,
			Type:  binary.BigEndian.Uint16(data[off : off+2]),
			Class: binary.BigEndian.Uint16(data[off+2 : off+4]),
		}
		off += 4
		msg.Questions = append(msg.Questions, q)
	}

	// Resource records
	var err error
	msg.Answers, off, err = unpackRRs(data, off, int(msg.Header.ANCount))
	if err != nil {
		return nil, fmt.Errorf("unpacking answers: %w", err)
	}
	msg.Authority, off, err = unpackRRs(data, off, int(msg.Header.NSCount))
	if err != nil {
		return nil, fmt.Errorf("unpacking authority: %w", err)
	}
	msg.Additional, _, err = unpackRRs(data, off, int(msg.Header.ARCount))
	if err != nil {
		return nil, fmt.Errorf("unpacking additional: %w", err)
	}

	return msg, nil
}

func unpackRRs(data []byte, off, count int) ([]ResourceRecord, int, error) {
	rrs := make([]ResourceRecord, 0, count)
	for range count {
		name, newOff, err := decodeName(data, off)
		if err != nil {
			return nil, off, fmt.Errorf("decoding RR name: %w", err)
		}
		off = newOff
		if off+10 > len(data) {
			return nil, off, errors.New("resource record truncated")
		}
		rr := ResourceRecord{
			Name:  name,
			Type:  binary.BigEndian.Uint16(data[off : off+2]),
			Class: binary.BigEndian.Uint16(data[off+2 : off+4]),
			TTL:   binary.BigEndian.Uint32(data[off+4 : off+8]),
		}
		rdLen := binary.BigEndian.Uint16(data[off+8 : off+10])
		off += 10
		if off+int(rdLen) > len(data) {
			return nil, off, errors.New("RDATA truncated")
		}
		rr.Data = make([]byte, rdLen)
		copy(rr.Data, data[off:off+int(rdLen)])
		off += int(rdLen)
		rrs = append(rrs, rr)
	}
	return rrs, off, nil
}

// encodeName converts a dotted DNS name to wire format (uncompressed).
// Input: "example.local." or "example.local" (trailing dot optional).
func encodeName(name string) ([]byte, error) {
	name = strings.TrimSuffix(name, ".")
	if name == "" {
		return []byte{0}, nil // root label
	}

	labels := strings.Split(name, ".")
	buf := make([]byte, 0, len(name)+2)
	for _, label := range labels {
		if len(label) == 0 {
			return nil, errors.New("empty label in name")
		}
		if len(label) > 63 {
			return nil, fmt.Errorf("label %q exceeds 63 bytes", label)
		}
		buf = append(buf, byte(len(label)))
		buf = append(buf, label...)
	}
	buf = append(buf, 0) // root terminator

	return buf, nil
}

// decodeName reads a DNS name from wire format, handling compression pointers.
// Returns the decoded name (dot-separated, with trailing dot) and the offset after the name.
func decodeName(data []byte, off int) (string, int, error) {
	labels := make([]string, 0, 8)
	visited := make(map[int]bool) // detect pointer loops
	endOff := -1                  // tracks where to resume after following pointers
	maxLabels := 128              // safety limit

	for range maxLabels {
		if off >= len(data) {
			return "", 0, errors.New("name extends beyond message")
		}

		length := int(data[off])

		if length == 0 {
			// Root terminator
			if endOff == -1 {
				endOff = off + 1
			}
			break
		}

		// Check for compression pointer (top 2 bits set)
		if length&0xC0 == 0xC0 {
			if off+1 >= len(data) {
				return "", 0, errors.New("pointer extends beyond message")
			}
			ptr := int(binary.BigEndian.Uint16(data[off:off+2]) & 0x3FFF)
			if visited[ptr] {
				return "", 0, errors.New("compression pointer loop")
			}
			visited[ptr] = true
			if endOff == -1 {
				endOff = off + 2 // first pointer sets the resume offset
			}
			off = ptr
			continue
		}

		// Regular label
		off++
		if off+length > len(data) {
			return "", 0, errors.New("label extends beyond message")
		}
		labels = append(labels, string(data[off:off+length]))
		off += length
	}

	if endOff == -1 {
		return "", 0, errors.New("name decoding exceeded max labels")
	}

	return strings.Join(labels, ".") + ".", endOff, nil
}

// RDATA builders

// NewARecord builds RDATA for a DNS A record (IPv4 address).
func NewARecord(ip net.IP) []byte {
	v4 := ip.To4()
	if v4 == nil {
		return nil
	}
	return []byte(v4)
}

// NewPTRRecord builds RDATA for a DNS PTR record.
func NewPTRRecord(target string) ([]byte, error) {
	return encodeName(target)
}

// NewSRVRecord builds RDATA for a DNS SRV record (RFC 2782).
func NewSRVRecord(priority, weight, port uint16, target string) ([]byte, error) {
	name, err := encodeName(target)
	if err != nil {
		return nil, err
	}
	buf := make([]byte, 6, 6+len(name))
	binary.BigEndian.PutUint16(buf[0:2], priority)
	binary.BigEndian.PutUint16(buf[2:4], weight)
	binary.BigEndian.PutUint16(buf[4:6], port)
	buf = append(buf, name...)
	return buf, nil
}

// NewTXTRecord builds RDATA for a DNS TXT record from key=value pairs.
func NewTXTRecord(kv map[string]string) []byte {
	if len(kv) == 0 {
		// Empty TXT record: single zero-length string
		return []byte{0}
	}

	buf := make([]byte, 0, len(kv)*16)
	for k, v := range kv {
		s := k + "=" + v
		if len(s) > 255 {
			s = s[:255]
		}
		buf = append(buf, byte(len(s)))
		buf = append(buf, s...)
	}
	return buf
}
