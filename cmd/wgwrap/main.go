// wgwrap runs a userspace WireGuard tunnel from a wg-quick style .conf file.
//
//   go run ./cmd/wgwrap -config ./my.conf
//
// Linux default interface name is wg0; macOS uses utun (auto); Windows uses Wintun with a custom name.
// wireguard-go does not assign [Interface] Address to the TUN — configure IP routes yourself (e.g. ip addr / ifconfig).
//
// SPDX: app glue MIT; underlying golang.zx2c4.com/wireguard is MIT / WireGuard LLC.
package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"

	"github.com/fluxorio/fluxor/cmd/wgwrap/wgconf"
	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/tun"
)

func defaultIface() string {
	switch runtime.GOOS {
	case "darwin":
		return "utun"
	case "windows":
		return "FluxorWG"
	default:
		return "wg0"
	}
}

func main() {
	configPath := flag.String("config", "", "path to wg-quick style .conf (required)")
	iface := flag.String("iface", defaultIface(), "TUN interface name")
	mtuFlag := flag.Int("mtu", 0, "override TUN MTU (0 = from [Interface] MTU or 1420)")
	verbose := flag.Bool("v", false, "verbose WireGuard logs")
	flag.Parse()

	if *configPath == "" {
		flag.Usage()
		os.Exit(2)
	}

	raw, err := os.ReadFile(filepath.Clean(*configPath))
	if err != nil {
		log.Fatalf("read config: %v", err)
	}

	parsed, err := wgconf.ParseQuick(string(raw))
	if err != nil {
		log.Fatalf("parse config: %v", err)
	}

	mtu := *mtuFlag
	if mtu <= 0 {
		if parsed.MTU > 0 {
			mtu = parsed.MTU
		} else {
			mtu = device.DefaultMTU
		}
	}

	tdev, err := tun.CreateTUN(*iface, mtu)
	if err != nil {
		log.Fatalf("CreateTUN: %v", err)
	}
	realName, err := tdev.Name()
	if err == nil && realName != "" {
		log.Printf("TUN device: %s (mtu=%d)", realName, mtu)
	} else {
		log.Printf("TUN mtu=%d", mtu)
	}

	logLevel := device.LogLevelError
	if *verbose {
		logLevel = device.LogLevelVerbose
	}
	logger := device.NewLogger(logLevel, "(wgwrap) ")

	dev := device.NewDevice(tdev, conn.NewDefaultBind(), logger)

	if err := dev.IpcSet(parsed.UAPI); err != nil {
		log.Fatalf("IpcSet: %v", err)
	}
	if err := dev.Up(); err != nil {
		log.Fatalf("Up: %v", err)
	}

	if len(parsed.AddressLines) > 0 {
		log.Printf("note: set interface address manually, e.g. %v on %s", parsed.AddressLines, realName)
	}

	log.Printf("WireGuard up; Ctrl+C to exit")

	term := make(chan os.Signal, 1)
	signal.Notify(term, os.Interrupt)
	<-term

	dev.Close()
	log.Printf("shutdown complete")
}
