// Copyright (c) 2024-2028 Fluxor Framework
// All rights reserved.
//
// This source code is proprietary and confidential.
// Unauthorized copying, modification, distribution, or use of this software,
// via any medium is strictly prohibited without the express written permission
// of Fluxor Framework.
//
// This code is provided as an example for demonstration purposes only.
// Redistribution or sharing of this source code is not permitted.
//
// License: Proprietary - All Rights Reserved
// For licensing inquiries, please contact: caokhang91@gmail.com

package main

import (
	"github.com/fluxorio/fluxor/pkg/lite/fluxor"
	"github.com/fluxorio/fluxor/pkg/lite/fx"
	"github.com/fluxorio/fluxor/pkg/lite/web"
)

func main() {
	// 1) Runtime
	app := fluxor.New()

	// Subscribe once (global)
	app.Bus().Subscribe("log", func(msg any) {
		// In a real service: structured logger, redaction, correlation IDs, etc.
		println("log:", msg)
	})

	// 2) Routes / logic
	r := web.NewRouter()

	r.GET("/api/ping", func(c *fx.Context) error {
		return c.Ok(fx.JSON{"msg": "pong", "framework": "Fluxor (lite)"})
	})

	r.GET("/api/work", func(c *fx.Context) error {
		// Demo: WorkerPool + Bus (avoid capturing request context into goroutine)
		wp := c.Worker()
		bus := c.Bus()
		wp.Submit(func() {
			bus.Publish("log", "Heavy Task Done")
		})

		return c.Ok(fx.JSON{"status": "processing"})
	})

	// 3) Component
	v := web.NewHttpVerticle("8080", r)

	// 4) Deploy & Run
	app.Deploy(v)
	app.Run()
}
