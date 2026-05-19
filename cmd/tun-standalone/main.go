// tun-standalone: chỉ tạo TUN và (tuỳ chọn) bật IP — không kết nối VPN server.
// Dùng để thử interface, route, hoặc gắn process khác vào cùng stack mạng sau này.
//
// macOS (cần sudo):
//   sudo go run ./cmd/tun-standalone -up
//   sudo ./bin/tun-standalone -tun-ip 10.9.0.1 -tun-gw 10.9.0.0 -up
//
// Linux (cần sudo, dùng lệnh ip):
//   sudo go run ./cmd/tun-standalone -up
//
// Thoát: Ctrl+C (interface biến mất khi process kết thúc).
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"

	"github.com/songgao/water"
)

func main() {
	tunIP := flag.String("tun-ip", "10.8.0.1", "Địa chỉ IP local trên TUN")
	tunGW := flag.String("tun-gw", "10.8.0.0", "Peer / gateway (point-to-point)")
	doUp := flag.Bool("up", false, "Tự cấu hình: macOS ifconfig / Linux ip addr + link set")
	quiet := flag.Bool("quiet", false, "Chỉ in tên interface rồi giữ im lặng")
	flag.Parse()

	cfg := water.Config{DeviceType: water.TUN}
	ifce, err := water.New(cfg)
	if err != nil {
		if strings.Contains(err.Error(), "operation not permitted") || strings.Contains(err.Error(), "permission denied") {
			log.Fatal("Cần quyền root để tạo TUN: sudo ./bin/tun-standalone -up")
		}
		log.Fatalf("water.New: %v", err)
	}
	defer ifce.Close()

	name := ifce.Name()
	if !*quiet {
		log.Printf("TUN: %s  local=%s peer=%s", name, *tunIP, *tunGW)
	} else {
		fmt.Println(name)
	}

	if *doUp {
		if err := bringUp(name, *tunIP, *tunGW); err != nil {
			log.Fatalf("bring up: %v", err)
		}
		if !*quiet {
			log.Print("Đã bật interface (up).")
		}
	} else if !*quiet {
		printManual(name, *tunIP, *tunGW)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	if !*quiet {
		log.Print("Đang giữ TUN (đọc và bỏ gói). Ctrl+C để thoát.")
	}

	go func() {
		<-sig
		_ = ifce.Close()
		os.Exit(0)
	}()

	buf := make([]byte, 65536)
	for {
		_, err := ifce.Read(buf)
		if err != nil {
			if err != io.EOF && !*quiet {
				log.Printf("read: %v", err)
			}
			return
		}
	}
}

func bringUp(iface, localIP, peerIP string) error {
	switch runtime.GOOS {
	case "darwin":
		cmd := exec.Command("ifconfig", iface, localIP, peerIP, "up")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	case "linux":
		// Point-to-point giống ifconfig trên macOS
		link := exec.Command("ip", "link", "set", iface, "up")
		link.Stdout = os.Stdout
		link.Stderr = os.Stderr
		if err := link.Run(); err != nil {
			return err
		}
		add := exec.Command("ip", "addr", "add", localIP, "peer", peerIP, "dev", iface)
		add.Stdout = os.Stdout
		add.Stderr = os.Stderr
		if err := add.Run(); err != nil {
			// Có thể đã có địa chỉ — thử tiếp
			log.Printf("ip addr add: %v (bỏ qua nếu đã cấu hình)", err)
		}
		return nil
	default:
		return fmt.Errorf("GOOS=%s: chỉ hỗ trợ darwin và linux; tự cấu hình tay cho %s", runtime.GOOS, iface)
	}
}

func printManual(iface, localIP, peerIP string) {
	switch runtime.GOOS {
	case "darwin":
		fmt.Printf("Chạy: sudo ifconfig %s %s %s up\n", iface, localIP, peerIP)
	case "linux":
		fmt.Printf("Chạy:\n  sudo ip link set %s up\n  sudo ip addr add %s peer %s dev %s\n", iface, localIP, peerIP, iface)
	default:
		fmt.Printf("Tự cấu hình interface: %s\n", iface)
	}
}
