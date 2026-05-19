// Sugar Dev Demo - Express.js-like syntax with Fluxor
// Only ~25 lines of code!
package main

import (
	"github.com/fluxorio/fluxor/pkg/sugar"
	"github.com/fluxorio/fluxor/pkg/web"
)

func main() {
	sugar.Run("sugardev", ":8080", func(r *sugar.Router) {

		r.GET("/", func(c *web.FastRequestContext) error {
			return c.JSON(200, sugar.JSON{
				"message": "Hello from Sugar Dev! 🍬",
				"pattern": "Express.js-like",
			})
		})

		r.GET("/ping", func(c *web.FastRequestContext) error {
			return c.JSON(200, sugar.JSON{"pong": true})
		})

		r.GET("/health", func(c *web.FastRequestContext) error {
			return c.JSON(200, sugar.JSON{
				"status": "healthy",
				"uptime": "running",
			})
		})

	})
}
