// vpn-tun-client: VPN client với TUN — kết nối tới pkg/vpn server, bridge traffic TUN ↔ VPN Data.
//
// Trên macOS tạo TUN bắt buộc phải chạy bằng sudo.
//
// TCP (local test):
//   go run ./cmd/vpn-server
//   sudo ./bin/vpn-tun-client -server 127.0.0.1:1194 -up
//
// UDP + STUN (194.233.73.36): Bước 1 mở kết nối STUN (UDP), bước 2 chạy VPN over UDP.
//   sudo ./bin/vpn-tun-client -proto udp -server 194.233.73.36:1194 -stun 194.233.73.36:3478 -up
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/fluxorio/fluxor/pkg/vpn"
	"github.com/pion/stun/v3"
	"github.com/songgao/water"
)

func main() {
	serverAddr := flag.String("server", "127.0.0.1:1194", "VPN server address (TCP hoặc UDP)")
	proto := flag.String("proto", "tcp", "Protocol: tcp hoặc udp")
	stunAddr := flag.String("stun", "", "STUN server (UDP), ví dụ 194.233.73.36:3478 — bước 1 trước khi VPN; chỉ dùng khi -proto=udp")
	username := flag.String("username", "test", "VPN username")
	password := flag.String("password", "test", "VPN password")
	tunIP := flag.String("tun-ip", "10.8.0.1", "TUN interface IP (phải trùng IP server cấp, thường client đầu tiên = 10.8.0.1)")
	tunGW := flag.String("tun-gw", "10.8.0.0", "TUN gateway (point-to-point remote)")
	doUp := flag.Bool("up", false, "Bring up TUN with ifconfig (macOS)")
	applyRoutes := flag.Bool("apply-routes", false, "Chạy route add cho từng route server push (cần sudo)")
	flag.Parse()

	if runtime.GOOS != "darwin" {
		log.Fatal("TUN client chỉ hỗ trợ macOS (darwin). Cần quyền root: sudo ...")
	}

	// 1. Tạo TUN (trên macOS cần sudo)
	config := water.Config{DeviceType: water.TUN}
	ifce, err := water.New(config)
	if err != nil {
		if strings.Contains(err.Error(), "operation not permitted") || strings.Contains(err.Error(), "permission denied") {
			log.Fatal("Tạo TUN cần quyền root. Chạy: sudo ./bin/vpn-tun-client -server ... (hoặc sudo go run ./cmd/vpn-tun-client ...)")
		}
		log.Fatalf("water.New: %v", err)
	}
	defer ifce.Close()
	name := ifce.Name()
	log.Printf("TUN: %s (IP %s -> GW %s)", name, *tunIP, *tunGW)

	if *doUp {
		cmd := exec.Command("ifconfig", name, *tunIP, *tunGW, "up")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			log.Printf("ifconfig failed: chạy thủ công: sudo ifconfig %s %s %s up", name, *tunIP, *tunGW)
		}
	} else {
		fmt.Printf("Chạy: sudo ifconfig %s %s %s up\n", name, *tunIP, *tunGW)
	}

	var conn io.ReadWriteCloser
reconnect:
	if conn != nil {
		if c, ok := conn.(io.Closer); ok {
			c.Close()
		}
		conn = nil
	}

	// 2. Kết nối VPN server: TCP hoặc UDP (bước 1 STUN nếu -stun, bước 2 VPN over UDP)
	switch *proto {
	case "tcp":
		c, err := net.DialTimeout("tcp", *serverAddr, 10*time.Second)
		if err != nil {
			log.Printf("Dial TCP %s: %v, retry sau 3s", *serverAddr, err)
			time.Sleep(3 * time.Second)
			goto reconnect
		}
		conn = c
		log.Printf("Connected TCP to %s", *serverAddr)
	case "udp":
		udpConn, err := connectUDP(*serverAddr, *stunAddr)
		if err != nil {
			log.Printf("UDP %s: %v, retry sau 3s", *serverAddr, err)
			time.Sleep(3 * time.Second)
			goto reconnect
		}
		conn = udpConn
		log.Printf("VPN over UDP to %s", *serverAddr)
	default:
		log.Fatalf("proto phải là tcp hoặc udp, got %q", *proto)
	}

	// 3. Handshake -> Auth -> Config
	seq := uint32(0)
	send := func(p *vpn.VPNPacket) {
		data, err := p.Serialize()
		if err != nil {
			log.Fatalf("Serialize: %v", err)
		}
		if _, err := conn.Write(data); err != nil {
			log.Fatalf("Write: %v", err)
		}
	}
	readResp := func() *vpn.VPNPacket {
		buf := make([]byte, 65535)
		if d, ok := conn.(interface{ SetReadDeadline(time.Time) error }); ok {
			d.SetReadDeadline(time.Now().Add(15 * time.Second))
		}
		n, err := conn.Read(buf)
		if err != nil {
			log.Fatalf("Read: %v", err)
		}
		p, err := vpn.ParsePacket(buf[:n])
		if err != nil {
			log.Fatalf("ParsePacket: %v", err)
		}
		seq = p.Sequence + 1
		return p
	}

	// Handshake
	send(vpn.CreateHandshakePacket(seq))
	readResp()
	log.Print("Handshake OK")

	// Auth (server trả salt 16 byte để derive key ChaCha20-Poly1305)
	authPkt, _ := vpn.CreateAuthPacket(*username, *password, seq)
	send(authPkt)
	authResp := readResp()
	log.Print("Auth OK")
	var clientCrypto *vpn.ClientCrypto
	if len(authResp.Data) >= 16 {
		var err error
		clientCrypto, err = vpn.NewClientCrypto(authResp.Data[:16], *password)
		if err != nil {
			log.Printf("NewClientCrypto: %v", err)
		} else {
			log.Print("ChaCha20-Poly1305 enabled")
		}
	}

	// Config (chuyển sang StateConnected)
	cfgPkt := &vpn.VPNPacket{
		Type:      vpn.PacketTypeControlConfig,
		Version:   1,
		Sequence:  seq,
		Timestamp: uint64(time.Now().UnixNano()),
		Data:      []byte("CONFIG"),
	}
	send(cfgPkt)
	cfgResp := readResp()
	log.Print("Config OK — bridge TUN ↔ VPN")
	if len(cfgResp.Data) > 0 {
		dns, routes := vpn.ParseConfigResponse(cfgResp.Data)
		if len(dns) > 0 {
			log.Printf("DNS push: %v", dns)
		}
		if len(routes) > 0 {
			log.Printf("Routes push: %v", routes)
			if *applyRoutes {
				applyRoutesOS(routes, *tunGW, name)
			}
		}
	}

	if d, ok := conn.(interface{ SetReadDeadline(time.Time) error }); ok {
		d.SetReadDeadline(time.Time{})
	}

	// 4. Bridge: TUN -> VPN và VPN -> TUN
	var wg sync.WaitGroup
	packet := make([]byte, 2000)
	dataSeq := uint32(0)
	errCh := make(chan error, 2)

	// TUN -> VPN (encrypt payload nếu có clientCrypto)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			n, err := ifce.Read(packet)
			if err != nil {
				errCh <- err
				return
			}
			dataSeq++
			payload := packet[:n]
			if clientCrypto != nil {
				enc, err := clientCrypto.Encrypt(payload)
				if err != nil {
					continue
				}
				payload = enc
			}
			p := vpn.CreateDataPacket(payload, dataSeq)
			data, _ := p.Serialize()
			if _, err := conn.Write(data); err != nil {
				errCh <- err
				return
			}
		}
	}()

	// VPN -> TUN
	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 65535)
		for {
			n, err := conn.Read(buf)
			if err != nil {
				errCh <- err
				return
			}
			p, err := vpn.ParsePacket(buf[:n])
			if err != nil {
				continue
			}
			if p.Type == vpn.PacketTypeData && len(p.Data) > 0 {
				payload := p.Data
				if clientCrypto != nil {
					plain, err := clientCrypto.Decrypt(payload)
					if err != nil {
						continue
					}
					payload = plain
				}
				if _, err := ifce.Write(payload); err != nil {
					errCh <- err
					return
				}
			}
		}
	}()

	<-errCh
	log.Printf("Mất kết nối, reconnect sau 3s...")
	time.Sleep(3 * time.Second)
	goto reconnect
}

// udpSession: VPN over UDP — gửi tới serverAddr, nhận từ cùng conn (server reply về địa chỉ client).
type udpSession struct {
	conn   *net.UDPConn
	server *net.UDPAddr
}

func (u *udpSession) Read(buf []byte) (int, error) {
	n, _, err := u.conn.ReadFromUDP(buf)
	return n, err
}

func (u *udpSession) Write(data []byte) (int, error) {
	return u.conn.WriteToUDP(data, u.server)
}

func (u *udpSession) Close() error {
	return u.conn.Close()
}

func (u *udpSession) SetReadDeadline(t time.Time) error {
	return u.conn.SetReadDeadline(t)
}

// connectUDP: Bước 1 (nếu stunAddr != "") gửi STUN Binding Request tới stunAddr để mở NAT;
// bước 2 dùng cùng UDP socket để nói VPN với serverAddr. Trả về session gửi/nhận tới server.
func connectUDP(serverAddr, stunAddr string) (*udpSession, error) {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		return nil, err
	}

	if stunAddr != "" {
		pubIP, pubPort, err := doSTUN(conn, stunAddr)
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("STUN %s: %w", stunAddr, err)
		}
		log.Printf("STUN OK — public %s:%d (kết nối UDP đã mở)", pubIP, pubPort)
	}

	server, err := net.ResolveUDPAddr("udp", serverAddr)
	if err != nil {
		conn.Close()
		return nil, err
	}

	return &udpSession{conn: conn, server: server}, nil
}

func doSTUN(conn *net.UDPConn, stunServer string) (net.IP, int, error) {
	addr, err := net.ResolveUDPAddr("udp", stunServer)
	if err != nil {
		return nil, 0, err
	}
	msg := stun.MustBuild(stun.TransactionID, stun.BindingRequest)
	msg.Encode()
	var buf bytes.Buffer
	if _, err := msg.WriteTo(&buf); err != nil {
		return nil, 0, err
	}
	if _, err := conn.WriteToUDP(buf.Bytes(), addr); err != nil {
		return nil, 0, err
	}
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	b := make([]byte, 1024)
	n, _, err := conn.ReadFromUDP(b)
	if err != nil {
		return nil, 0, err
	}
	if !stun.IsMessage(b[:n]) {
		return nil, 0, fmt.Errorf("not a STUN response")
	}
	resp := stun.New()
	if _, err := resp.ReadFrom(bytes.NewReader(b[:n])); err != nil {
		return nil, 0, err
	}
	var xorAddr stun.XORMappedAddress
	if err := xorAddr.GetFrom(resp); err != nil {
		return nil, 0, err
	}
	return xorAddr.IP, xorAddr.Port, nil
}

// applyRoutesOS chạy route add cho từng CIDR.
// macOS: dùng -interface <utunN> — không dùng -tun-gw (10.8.0.0) làm gateway vì dễ gặp
// "Can't assign requested address" (network address không hợp lệ làm next-hop).
// Linux: ip route add ... dev <iface>.
func applyRoutesOS(routes []string, gateway, ifaceName string) {
	for _, r := range routes {
		_, ipNet, err := net.ParseCIDR(r)
		if err != nil {
			log.Printf("Parse route %q: %v", r, err)
			continue
		}
		if len(ipNet.Mask) != 4 {
			continue
		}
		maskStr := fmt.Sprintf("%d.%d.%d.%d", ipNet.Mask[0], ipNet.Mask[1], ipNet.Mask[2], ipNet.Mask[3])
		netStr := ipNet.IP.String()
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			// route add -net 10.8.0.0 -netmask 255.255.255.0 -interface utun6
			cmd = exec.Command("route", "add", "-net", netStr, "-netmask", maskStr, "-interface", ifaceName)
		case "linux":
			cmd = exec.Command("ip", "route", "add", r, "dev", ifaceName)
		default:
			log.Printf("apply-routes chưa hỗ trợ %s", runtime.GOOS)
			return
		}
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			log.Printf("route add %s: %v (có thể đã tồn tại)", r, err)
		} else {
			if runtime.GOOS == "darwin" {
				log.Printf("Route added: %s dev %s", r, ifaceName)
			} else {
				log.Printf("Route added: %s via %s (dev %s)", r, gateway, ifaceName)
			}
		}
	}
}
