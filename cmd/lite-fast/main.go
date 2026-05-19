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
	"github.com/fluxorio/fluxor/pkg/lite/webfast"
)

func main() {
	app := fluxor.New()

	// Use the component-scoped core context for logs, bus, worker (bound in verticle start)
	router := webfast.NewRouter()

	// For 500k RPS testing, keep handler extremely small and avoid JSON.
	router.GET("/ping",
		func(c *fx.FastContext) error {
			return c.Text(200, "pong")
		},
		webfast.Cache(webfast.CacheConfig{
			CacheControl: "public, max-age=60, immutable",
			ETag:         `"ping-v1"`,
		}),
	)

	router.GET("/users/:id", func(c *fx.FastContext) error {
		return c.Text(200, c.Param("id"))
	})

	v := webfast.NewFastHTTPVerticle(":8080", router)
	app.Deploy(v)
	app.Run()
}
