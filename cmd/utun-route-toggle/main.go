// utun-route-toggle: bật/tắt traffic IPv4 tới một CIDR qua interface TUN (vd utun6).
//
// Trên macOS thực hiện bằng route add/delete (cần sudo), giống -apply-routes của vpn-tun-client.
//
//   go build -o bin/utun-route-toggle ./cmd/utun-route-toggle
//
//   sudo ./bin/utun-route-toggle on -iface utun6 -cidr 10.8.0.0/24
//   sudo ./bin/utun-route-toggle off -iface utun6 -cidr 10.8.0.0/24
//   ./bin/utun-route-toggle status -iface utun6
//
// "off" chỉ xóa route tới CIDR đã add bằng lệnh tương ứng; không đóng vpn-tun-client.
package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

func main() {
	log.SetFlags(0)

	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	sub := os.Args[1]
	args := os.Args[2:]
	if sub == "-h" || sub == "help" {
		usage()
		return
	}

	fs := flag.NewFlagSet(sub, flag.ExitOnError)
	iface := fs.String("iface", "utun6", "Tên interface (utun6, utun7, …)")
	cidr := fs.String("cidr", "10.8.0.0/24", "CIDR IPv4 cần đẩy qua TUN")
	_ = fs.Parse(args)

	switch sub {
	case "on":
		if err := routeOn(*iface, *cidr); err != nil {
			log.Fatal(err)
		}
		log.Printf("ON: %s → dev %s", *cidr, *iface)
	case "off":
		if err := routeOff(*cidr); err != nil {
			log.Fatal(err)
		}
		log.Printf("OFF: đã xóa route %s (nếu tồn tại)", *cidr)
	case "status":
		if err := status(*iface); err != nil {
			log.Fatal(err)
		}
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `Usage:
  sudo %s on  [-iface utun6] [-cidr 10.8.0.0/24]
  sudo %s off [-cidr 10.8.0.0/24]
  %s status [-iface utun6]

on  — route add -net … -netmask … -interface <iface>
off — route delete -net … -netmask …
status — in các dòng netstat có chứa iface (không cần sudo)

`, os.Args[0], os.Args[0], os.Args[0])
}

func parseIPv4Net(cidr string) (netStr, maskStr string, err error) {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", "", err
	}
	if len(ipNet.Mask) != 4 {
		return "", "", fmt.Errorf("chỉ hỗ trợ IPv4: %s", cidr)
	}
	ip4 := ipNet.IP.To4()
	if ip4 == nil {
		return "", "", fmt.Errorf("chỉ hỗ trợ IPv4: %s", cidr)
	}
	m := ipNet.Mask
	maskStr = fmt.Sprintf("%d.%d.%d.%d", m[0], m[1], m[2], m[3])
	return ip4.String(), maskStr, nil
}

func ifaceExists(name string) bool {
	ifaces, err := net.Interfaces()
	if err != nil {
		return false
	}
	for _, ifi := range ifaces {
		if ifi.Name == name {
			return true
		}
	}
	return false
}

func routeOn(iface, cidr string) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("on: hiện chỉ hỗ trợ darwin (macOS)")
	}
	if !ifaceExists(iface) {
		return fmt.Errorf("không thấy interface %q — bật vpn-tun-client (-up) trước", iface)
	}
	netStr, maskStr, err := parseIPv4Net(cidr)
	if err != nil {
		return err
	}
	cmd := exec.Command("route", "add", "-net", netStr, "-netmask", maskStr, "-interface", iface)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("route add: %w (thử sudo)", err)
	}
	return nil
}

func routeOff(cidr string) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("off: hiện chỉ hỗ trợ darwin (macOS)")
	}
	netStr, maskStr, err := parseIPv4Net(cidr)
	if err != nil {
		return err
	}
	cmd := exec.Command("route", "delete", "-net", netStr, "-netmask", maskStr)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("route delete: %w (có thể route đã không tồn tại; thử sudo)", err)
	}
	return nil
}

func status(iface string) error {
	out, err := exec.Command("netstat", "-rn", "-f", "inet").Output()
	if err != nil {
		return fmt.Errorf("netstat: %w", err)
	}
	lines := strings.Split(string(out), "\n")
	found := false
	for _, line := range lines {
		if strings.Contains(line, iface) {
			fmt.Println(line)
			found = true
		}
	}
	if !found {
		log.Printf("Không có dòng nào chứa %q trong netstat -rn -f inet", iface)
	}
	return nil
}
