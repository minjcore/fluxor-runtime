// vpn-server: chạy VPN server (pkg/vpn) để test với vpn-tun-client.
//
//   go run ./cmd/vpn-server           # TCP :1194
//   go run ./cmd/vpn-server -udp     # UDP :1194 (cho client STUN + VPN over UDP, ví dụ 194.233.73.36)
package main

import (
	"context"
	"flag"
	"log"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/vpn"
)

func main() {
	useUDP := flag.Bool("udp", false, "Listen UDP thay vì TCP")
	forward := flag.Bool("forward", false, "Bật gateway TUN, forward traffic dst ngoài pool ra Internet (chỉ Linux, cần ip_forward=1)")
	flag.Parse()

	cfg := vpn.DefaultConfig()
	cfg.ListenAddr = ":1194"
	cfg.NetworkCIDR = "10.8.0.0/24"
	cfg.EchoData = true // dev/test: TUN client nhận lại gói Data (round-trip)
	cfg.EnableForwarding = *forward
	if *forward {
		cfg.GatewayTUNIP = "10.8.0.254"
	}
	if *useUDP {
		cfg.Protocol = "udp"
	} else {
		cfg.Protocol = "tcp"
	}
	
	gocmd := core.NewGoCMD(context.Background())
	srv, err := vpn.NewVPNServer(gocmd, cfg)
	if err != nil {
		log.Fatalf("NewVPNServer: %v", err)
	}
	if err := srv.Start(); err != nil {
		log.Fatalf("Start: %v", err)
	}
	log.Printf("VPN server %s :1194 (EchoData=true). Client: sudo ./vpn-tun-client -proto %s -server <IP>:1194 -stun 194.233.73.36:3478 -up", cfg.Protocol, cfg.Protocol)

	select {} // block
}
