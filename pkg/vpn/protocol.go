package vpn

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
	"time"
)

// PacketType represents the type of VPN packet
type PacketType uint8

const (
	// Control packets
	PacketTypeControlHandshake PacketType = 0x01
	PacketTypeControlAuth       PacketType = 0x02
	PacketTypeControlConfig     PacketType = 0x03
	PacketTypeControlKeepAlive  PacketType = 0x04
	PacketTypeControlDisconnect PacketType = 0x05

	// Data packets
	PacketTypeData PacketType = 0x10
	PacketTypeACK  PacketType = 0x11
)

// ConnectionState represents the state of a VPN connection
type ConnectionState uint8

const (
	StateInitial      ConnectionState = 0
	StateHandshaking  ConnectionState = 1
	StateAuthenticating ConnectionState = 2
	StateConfiguring   ConnectionState = 3
	StateConnected    ConnectionState = 4
	StateDisconnected ConnectionState = 5
)

// VPNPacket represents a VPN protocol packet
type VPNPacket struct {
	Type      PacketType
	Version   uint8
	Length    uint16
	Sequence  uint32
	Timestamp uint64
	Data      []byte
}

// ParsePacket parses a VPN packet from raw bytes
func ParsePacket(data []byte) (*VPNPacket, error) {
	if len(data) < 16 {
		return nil, fmt.Errorf("packet too short: %d bytes", len(data))
	}

	packet := &VPNPacket{
		Type:      PacketType(data[0]),
		Version:   data[1],
		Length:    binary.BigEndian.Uint16(data[2:4]),
		Sequence:  binary.BigEndian.Uint32(data[4:8]),
		Timestamp: binary.BigEndian.Uint64(data[8:16]),
	}

	// Validate length
	if int(packet.Length) != len(data) {
		return nil, fmt.Errorf("packet length mismatch: expected %d, got %d", packet.Length, len(data))
	}

	// Extract payload
	if len(data) > 16 {
		packet.Data = make([]byte, len(data)-16)
		copy(packet.Data, data[16:])
	}

	return packet, nil
}

// Serialize serializes a VPN packet to bytes
func (p *VPNPacket) Serialize() ([]byte, error) {
	if len(p.Data) > 65535-16 {
		return nil, fmt.Errorf("packet data too large: %d bytes", len(p.Data))
	}

	length := uint16(16 + len(p.Data))
	buf := make([]byte, length)

	buf[0] = byte(p.Type)
	buf[1] = p.Version
	binary.BigEndian.PutUint16(buf[2:4], length)
	binary.BigEndian.PutUint32(buf[4:8], p.Sequence)
	binary.BigEndian.PutUint64(buf[8:16], p.Timestamp)

	if len(p.Data) > 0 {
		copy(buf[16:], p.Data)
	}

	return buf, nil
}

// CreateHandshakePacket creates a handshake packet
func CreateHandshakePacket(sequence uint32) *VPNPacket {
	return &VPNPacket{
		Type:      PacketTypeControlHandshake,
		Version:   1,
		Sequence:  sequence,
		Timestamp: uint64(time.Now().UnixNano()),
		Data:      []byte("HANDSHAKE"),
	}
}

// CreateAuthResponseWithSalt tạo auth response với salt 16 byte (để client derive key).
func CreateAuthResponseWithSalt(sequence uint32, salt []byte) *VPNPacket {
	return &VPNPacket{
		Type:      PacketTypeControlAuth,
		Version:   1,
		Sequence:  sequence,
		Timestamp: uint64(time.Now().UnixNano()),
		Data:      salt,
	}
}

// CreateAuthPacket creates an authentication packet
func CreateAuthPacket(username, password string, sequence uint32) (*VPNPacket, error) {
	// Simple auth packet format: username:password
	authData := fmt.Sprintf("%s:%s", username, password)
	packet := &VPNPacket{
		Type:      PacketTypeControlAuth,
		Version:   1,
		Sequence:  sequence,
		Timestamp: uint64(time.Now().UnixNano()),
		Data:      []byte(authData),
	}
	return packet, nil
}

// CreateKeepAlivePacket creates a keep-alive packet
func CreateKeepAlivePacket(sequence uint32) *VPNPacket {
	return &VPNPacket{
		Type:      PacketTypeControlKeepAlive,
		Version:   1,
		Sequence:  sequence,
		Timestamp: uint64(time.Now().UnixNano()),
		Data:      []byte("KEEPALIVE"),
	}
}

// CreateDataPacket creates a data packet
func CreateDataPacket(data []byte, sequence uint32) *VPNPacket {
	return &VPNPacket{
		Type:      PacketTypeData,
		Version:   1,
		Sequence:  sequence,
		Timestamp: uint64(time.Now().UnixNano()),
		Data:      data,
	}
}

// CreateDisconnectPacket creates a disconnect packet
func CreateDisconnectPacket(sequence uint32, reason string) *VPNPacket {
	return &VPNPacket{
		Type:      PacketTypeControlDisconnect,
		Version:   1,
		Sequence:  sequence,
		Timestamp: uint64(time.Now().UnixNano()),
		Data:      []byte(reason),
	}
}

// CreateACKPacket creates an ACK packet
func CreateACKPacket(sequence uint32) *VPNPacket {
	return &VPNPacket{
		Type:      PacketTypeACK,
		Version:   1,
		Sequence:  sequence,
		Timestamp: uint64(time.Now().UnixNano()),
		Data:      []byte("ACK"),
	}
}

// CreateConfigResponsePacket creates config response with DNS and routes (push to client).
// Format Data: "DNS=8.8.8.8,8.8.4.4 ROUTES=10.8.0.0/24"
func CreateConfigResponsePacket(sequence uint32, dnsServers, routes []string) *VPNPacket {
	var dns, rts string
	for i, s := range dnsServers {
		if i > 0 {
			dns += ","
		}
		dns += s
	}
	for i, s := range routes {
		if i > 0 {
			rts += ","
		}
		rts += s
	}
	data := fmt.Sprintf("DNS=%s ROUTES=%s", dns, rts)
	return &VPNPacket{
		Type:      PacketTypeControlConfig,
		Version:   1,
		Sequence:  sequence,
		Timestamp: uint64(time.Now().UnixNano()),
		Data:      []byte(data),
	}
}

// ParseConfigResponse parses DNS and routes from config response Data.
func ParseConfigResponse(data []byte) (dns []string, routes []string) {
	s := string(data)
	for _, part := range splitAndTrim(s, " ") {
		if strings.HasPrefix(part, "DNS=") && len(part) > 4 {
			for _, ip := range splitAndTrim(part[4:], ",") {
				if ip != "" {
					dns = append(dns, ip)
				}
			}
		}
		if strings.HasPrefix(part, "ROUTES=") && len(part) > 7 {
			for _, r := range splitAndTrim(part[7:], ",") {
				if r != "" {
					routes = append(routes, r)
				}
			}
		}
	}
	return dns, routes
}

func splitAndTrim(s, sep string) []string {
	var out []string
	for _, p := range bytes.Split([]byte(s), []byte(sep)) {
		out = append(out, strings.TrimSpace(string(p)))
	}
	return out
}

// ParseAuthData parses authentication data from a packet
func ParseAuthData(packet *VPNPacket) (username, password string, err error) {
	if packet.Type != PacketTypeControlAuth {
		return "", "", fmt.Errorf("not an auth packet")
	}

	parts := bytes.Split(packet.Data, []byte(":"))
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid auth packet format")
	}

	return string(parts[0]), string(parts[1]), nil
}

// IsControlPacket checks if a packet is a control packet
func IsControlPacket(packet *VPNPacket) bool {
	return packet.Type < PacketTypeData
}

// IsDataPacket checks if a packet is a data packet
func IsDataPacket(packet *VPNPacket) bool {
	return packet.Type >= PacketTypeData
}
