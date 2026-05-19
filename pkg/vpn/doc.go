// Package vpn provides a VPN server for Fluxor.
//
// This package follows Fluxor's Premium Pattern architecture with BaseComponent
// lifecycle management, BaseConfig inheritance, and EventBus integration.
//
// Features:
//   - VPN protocol support (OpenVPN-compatible, WireGuard-style)
//   - Client authentication and management
//   - Network tunneling and routing
//   - Encryption and security
//   - Connection pooling and metrics
//   - Context support for cancellation and timeouts
//   - EventBus integration for reactive patterns
//   - Fail-fast validation throughout
//
// Quick Start:
//
//	package main
//
//	import (
//	    "github.com/fluxorio/fluxor/pkg/vpn"
//	    "github.com/fluxorio/fluxor/pkg/core"
//	)
//
//	type MyVerticle struct {
//	    *core.BaseVerticle
//	    vpnServer *vpn.VPNServer
//	}
//
//	func (v *MyVerticle) Start(ctx core.FluxorContext) error {
//	    // Create VPN server
//	    config := vpn.DefaultConfig()
//	    config.ListenAddr = ":1194"
//	    config.Protocol = "udp"
//	    config.NetworkCIDR = "10.8.0.0/24"
//	    v.vpnServer = vpn.NewVPNServer(ctx.GoCMD(), config)
//
//	    // Start VPN server
//	    if err := v.vpnServer.Start(); err != nil {
//	        return err
//	    }
//
//	    return nil
//	}
//
// For complete documentation and examples, see README.md.
//
// Path: pkg/vpn
package vpn
