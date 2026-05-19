// Package proxy provides a high-performance proxy server for Fluxor.
//
// This package follows Fluxor's Premium Pattern architecture with BaseComponent
// lifecycle management, BaseConfig inheritance, and EventBus integration.
//
// Features:
//   - HTTP and TCP proxy support
//   - Connection pooling and load balancing
//   - Health checking and failover
//   - Rate limiting and backpressure control
//   - Context support for cancellation and timeouts
//   - EventBus integration for reactive patterns
//   - Fail-fast validation throughout
//   - Metrics and observability
//
// Quick Start:
//
//	package main
//
//	import (
//	    "github.com/fluxorio/fluxor/pkg/proxy"
//	    "github.com/fluxorio/fluxor/pkg/core"
//	)
//
//	type MyVerticle struct {
//	    *core.BaseVerticle
//	    proxyServer *proxy.ProxyServer
//	}
//
//	func (v *MyVerticle) Start(ctx core.FluxorContext) error {
//	    // Create proxy server
//	    config := proxy.DefaultConfig()
//	    config.ListenAddr = ":8080"
//	    config.Backends = []proxy.Backend{
//	        {URL: "http://localhost:3000"},
//	        {URL: "http://localhost:3001"},
//	    }
//	    v.proxyServer = proxy.NewProxyServer(ctx.GoCMD(), config)
//
//	    // Start proxy server
//	    if err := v.proxyServer.Start(); err != nil {
//	        return err
//	    }
//
//	    return nil
//	}
//
// For complete documentation and examples, see README.md.
//
// Path: pkg/proxy
package proxy
