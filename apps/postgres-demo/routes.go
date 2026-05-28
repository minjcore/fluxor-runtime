package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fluxorio/fluxor/apps/postgres-demo/controllers"
	"github.com/fluxorio/fluxor/apps/postgres-demo/services"
	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/dbruntime"
	"github.com/fluxorio/fluxor/pkg/web"
	"github.com/fluxorio/fluxor/pkg/web/dashboard"
	"github.com/fluxorio/fluxor/pkg/web/middleware/auth"
	"github.com/valyala/fasthttp"
)

// maskAuthHeader masks the token in the Authorization header for logging
func maskAuthHeader(header string) string {
	if header == "" {
		return "(empty)"
	}
	if len(header) > 20 {
		return header[:10] + "..." + header[len(header)-10:]
	}
	return header
}

// Default credentials for demo
const (
	defaultUsername = "admin"
	defaultPassword = "admin123"
	jwtSecret       = "postgres-secret-key-change-in-production"
)

// setupWebUI sets up the HTTP web server with MVC architecture
func setupWebUI(gocmd core.GoCMD, db *dbruntime.DB, purchaseService *services.PurchaseService, walletService *services.WalletService, authService *services.AuthService, verticle *PostgresDemoVerticle, tlsCertFile, tlsKeyFile string) (*web.FastHTTPServer, error) {
	// Create HTTP server config
	addr := ":8081"
	cfg := web.DefaultFastHTTPServerConfig(addr)
	if tlsCertFile != "" && tlsKeyFile != "" {
		tlsCfg, err := web.NewTLSConfigFromFiles(tlsCertFile, tlsKeyFile)
		if err != nil {
			return nil, fmt.Errorf("TLS config: %w", err)
		}
		cfg.TLSConfig = tlsCfg
	}
	server := web.NewFastHTTPServer(gocmd, cfg)
	router := server.FastRouter()

	// Register metrics endpoint
	server.RegisterMetricsEndpoint()

	// Register dashboard routes (/api/dashboard/metrics, /api/dashboard/health).
	// This makes the load-test monitor framework work against this server.
	dashboard.Register(router, "")
	dashboard.GetMetricsCollector().RegisterHTTPServer("postgres-demo", &serverMetricsAdapter{server})

	// Note: authService and walletService are already initialized and configured in main.go
	// authService is passed in and already has database and walletService configured

	// Initialize services
	ledgerService := services.NewLedgerService(db)

	// Initialize wallet rule service
	walletRuleService := services.NewWalletRuleService(db)

	// Initialize system setting service
	systemSettingService := services.NewSystemSettingService(db)

	// Initialize approval service
	approvalService := services.NewApprovalService(db)

	// Initialize controllers
	pageController := controllers.NewPageController()
	authController := controllers.NewAuthController(authService)
	dashboardController := controllers.NewDashboardController()
	purchaseController := controllers.NewPurchaseController(purchaseService)
	walletController := controllers.NewWalletController(walletService)
	accountantController := controllers.NewAccountantController(ledgerService)
	walletRuleController := controllers.NewWalletRuleController(walletRuleService)
	systemSettingController := controllers.NewSystemSettingController(systemSettingService)
	approvalController := controllers.NewApprovalController(approvalService)

	// FastJWT middleware — 2.7× faster than ParseWithClaims, 85% fewer allocations.
	// Checks Authorization: Bearer <token> header first, then "token" cookie (browser).
	jwtMiddleware := auth.FastJWT(auth.FastJWTConfig{
		Secret:    jwtSecret,
		ClaimsKey: "user",
		OnError: func(ctx *web.FastRequestContext, err error) error {
			path := string(ctx.Path())
			if strings.HasPrefix(path, "/api/") {
				ctx.RequestCtx.SetStatusCode(401)
				ctx.RequestCtx.SetContentType("application/json")
				ctx.RequestCtx.WriteString(`{"error":"unauthorized","message":"invalid or missing token"}`)
				return nil
			}
			ctx.RequestCtx.Redirect("/login", 302)
			return nil
		},
	})

	// Static file server for public directory
	publicDir := "./public"
	absPublicDir, _ := filepath.Abs(publicDir)
	if _, err := os.Stat(publicDir); err == nil {
		fs := &fasthttp.FS{
			Root:               absPublicDir,
			IndexNames:         []string{"index.html"},
			GenerateIndexPages: false,
			Compress:           true,
			AcceptByteRange:    true,
		}
		fileHandler := fs.NewRequestHandler()
		
		// Serve static files from /public/* path
		router.GETFast("/public/*", func(ctx *web.FastRequestContext) error {
			// Get the requested path
			requestPath := string(ctx.Path())
			
			// Remove /public prefix to get the file path relative to public directory
			if strings.HasPrefix(requestPath, "/public/") {
				filePath := requestPath[8:] // Remove "/public/"
				if filePath == "" {
					filePath = "index.html"
				}
				
				// Set the path in RequestCtx to serve from public directory root
				ctx.RequestCtx.URI().SetPath("/" + filePath)
			}
			
			// Serve the file
			fileHandler(ctx.RequestCtx)
			return nil
		})
		
		log.Printf("Static file server enabled for directory: %s (accessible at /public/*)", absPublicDir)
	} else {
		log.Printf("Public directory not found: %s (static files will not be served)", publicDir)
	}

	// Public routes (no authentication required)
	router.GETFast("/", pageController.ShowLogin)
	router.GETFast("/login", pageController.ShowLogin)
	router.GETFast("/register", pageController.ShowRegister)

	// Auth API routes (no authentication required - these are public)
	// Note: These endpoints are intentionally public for login/registration
	router.POSTFast("/api/auth/login", authController.Login)
	router.POSTFast("/api/auth/logout", authController.Logout)
	router.POSTFast("/api/auth/register", authController.Register)

	// Public API endpoints (no authentication required)
	router.GETFast("/api/public/health", func(c *web.FastRequestContext) error {
		health := map[string]interface{}{
			"status":    "ok",
			"timestamp": time.Now().Unix(),
			"version":   "1.0.0",
		}

		if db != nil {
			stats := db.Stats()
			health["database"] = map[string]interface{}{
				"connected": stats.OpenConnections > 0,
				"in_use":    stats.InUse,
				"idle":      stats.Idle,
			}
		}

		return c.JSON(200, health)
	})

	// Public metrics endpoint (basic metrics only, no sensitive data)
	router.GETFast("/api/public/metrics", func(c *web.FastRequestContext) error {
		metrics := server.Metrics()
		publicMetrics := map[string]interface{}{
			"status": map[string]interface{}{
				"up": true,
			},
			"requests": map[string]interface{}{
				"total":      metrics.TotalRequests,
				"successful": metrics.SuccessfulRequests,
				"errors":     metrics.ErrorRequests,
			},
			"performance": map[string]interface{}{
				"average_latency_ms": fmt.Sprintf("%.2f", metrics.AverageLatencyMs),
				"arrival_rate":       fmt.Sprintf("%.2f", metrics.ArrivalRate),
			},
		}

		// Include Little's Law metrics
		if metrics.ExpectedQueueLength > 0 {
			publicMetrics["littles_law"] = map[string]interface{}{
				"expected_queue_length": fmt.Sprintf("%.2f", metrics.ExpectedQueueLength),
				"validation_ratio":      fmt.Sprintf("%.2f", metrics.LittlesLawValidation),
			}
		}

		return c.JSON(200, publicMetrics)
	})

	// Public server info endpoint
	router.GETFast("/api/public/info", func(c *web.FastRequestContext) error {
		metrics := server.Metrics()
		info := map[string]interface{}{
			"name":      "postgres-demo",
			"version":   "1.0.0",
			"status":    "running",
			"timestamp": time.Now().Unix(),
			"server": map[string]interface{}{
				"workers":        metrics.Workers,
				"queue_capacity": metrics.QueueCapacity,
			},
		}

		return c.JSON(200, info)
	})

	// Protected routes (require authentication)
	// Dashboard routes - protected with JWT middleware
	router.GETFastWith("/dashboard", dashboardController.ShowDashboard, jwtMiddleware)
	router.GETFastWith("/accountant", accountantController.ShowAccountantDashboard, jwtMiddleware)

	// Protected API routes (require authentication)
	// Apply both API protection and JWT middleware
	router.GETFastWith("/api/db/status", func(c *web.FastRequestContext) error {
		if db == nil {
			return c.JSON(500, map[string]interface{}{
				"error": "database not initialized",
			})
		}

		stats := db.Stats()

		return c.JSON(200, map[string]interface{}{
			"status":            "running",
			"open_connections": stats.OpenConnections,
			"in_use":            stats.InUse,
			"idle":              stats.Idle,
			"wait_count":        stats.WaitCount,
			"wait_duration":     stats.WaitDuration.String(),
		})
	}, jwtMiddleware)

	router.GETFastWith("/api/db/stats", func(c *web.FastRequestContext) error {
		if db == nil {
			return c.JSON(500, map[string]interface{}{
				"error": "database not initialized",
			})
		}

		stats := db.Stats()
		return c.JSON(200, map[string]interface{}{
			"open_connections": stats.OpenConnections,
			"in_use":            stats.InUse,
			"idle":              stats.Idle,
			"wait_count":        stats.WaitCount,
			"wait_duration":     stats.WaitDuration.String(),
		})
	}, jwtMiddleware)

	router.POSTFastWith("/api/db/query", func(c *web.FastRequestContext) error {
		if db == nil {
			return c.JSON(500, map[string]interface{}{
				"error": "database not initialized",
			})
		}

		var req struct {
			Query string `json:"query"`
		}
		if err := c.BindJSON(&req); err != nil {
			return c.JSON(400, map[string]interface{}{
				"error": "invalid request",
			})
		}

		// Execute query
		rows, err := db.QueryContext(c.Context(), req.Query)
		if err != nil {
			return c.JSON(500, map[string]interface{}{
				"error": err.Error(),
			})
		}
		defer rows.Close()

		// Get column names
		columns, err := rows.Columns()
		if err != nil {
			return c.JSON(500, map[string]interface{}{
				"error": err.Error(),
			})
		}

		// Scan rows
		var results []map[string]interface{}
		for rows.Next() {
			values := make([]interface{}, len(columns))
			valuePtrs := make([]interface{}, len(columns))
			for i := range values {
				valuePtrs[i] = &values[i]
			}

			if err := rows.Scan(valuePtrs...); err != nil {
				return c.JSON(500, map[string]interface{}{
					"error": err.Error(),
				})
			}

			row := make(map[string]interface{})
			for i, col := range columns {
				val := values[i]
				if b, ok := val.([]byte); ok {
					row[col] = string(b)
				} else {
					row[col] = val
				}
			}
			results = append(results, row)
		}

		if err := rows.Err(); err != nil {
			return c.JSON(500, map[string]interface{}{
				"error": err.Error(),
			})
		}

		return c.JSON(200, map[string]interface{}{
			"columns": columns,
			"rows":    results,
			"count":   len(results),
		})
	}, jwtMiddleware)

	// Purchase API routes (require authentication)
	if purchaseService != nil {
		router.GETFastWith("/api/products", purchaseController.GetProducts, jwtMiddleware)
		router.POSTFastWith("/api/purchase", purchaseController.MakePurchase, jwtMiddleware)
		router.GETFastWith("/api/orders", purchaseController.GetOrders, jwtMiddleware)
	}

	// Wallet API routes (require authentication)
	if purchaseService != nil {
		router.GETFastWith("/api/wallet/balance", walletController.GetBalance, jwtMiddleware)
		router.GETFastWith("/api/wallet/transactions", walletController.GetTransactions, jwtMiddleware)
		router.POSTFastWith("/api/wallet/add", walletController.AddBalance, jwtMiddleware)
		router.POSTFastWith("/api/wallet/transfer", walletController.Transfer, jwtMiddleware)
	}

	// Accountant API routes (require authentication)
	router.GETFastWith("/api/accountant/trial-balance", accountantController.GetTrialBalance, jwtMiddleware)
	router.GETFastWith("/api/accountant/account-balances", accountantController.GetAccountBalances, jwtMiddleware)
	router.GETFastWith("/api/accountant/journals", accountantController.GetJournals, jwtMiddleware)
	router.GETFastWith("/api/accountant/journals/:id", accountantController.GetJournalWithEntries, jwtMiddleware)
	router.GETFastWith("/api/accountant/accounts", accountantController.GetAccounts, jwtMiddleware)

	// Wallet Rules API routes (require authentication)
	router.GETFastWith("/api/wallet-rules", walletRuleController.GetAllRules, jwtMiddleware)
	router.POSTFastWith("/api/wallet-rules", walletRuleController.CreateRule, jwtMiddleware)
	router.POSTFastWith("/api/wallet-rules/:id/update", walletRuleController.UpdateRule, jwtMiddleware) // Using POST for update
	router.POSTFastWith("/api/wallet-rules/:id/delete", walletRuleController.DeleteRule, jwtMiddleware) // Using POST for delete

	// System Settings API routes (require authentication)
	router.GETFastWith("/api/settings", systemSettingController.GetAllSettings, jwtMiddleware)
	router.GETFastWith("/api/settings/:key", systemSettingController.GetSetting, jwtMiddleware)
	router.GETFastWith("/api/settings/:key/value", systemSettingController.GetSettingValue, jwtMiddleware)
	router.POSTFastWith("/api/settings", systemSettingController.CreateSetting, jwtMiddleware)
	router.POSTFastWith("/api/settings/:key/update", systemSettingController.UpdateSetting, jwtMiddleware)
	router.POSTFastWith("/api/settings/:key/delete", systemSettingController.DeleteSetting, jwtMiddleware)

	// Approval API routes (require authentication)
	router.GETFastWith("/api/approvals", approvalController.GetPendingApprovals, jwtMiddleware)
	router.GETFastWith("/api/approvals/:id", approvalController.GetApproval, jwtMiddleware)
	router.POSTFastWith("/api/approvals", approvalController.CreateApproval, jwtMiddleware)
	router.POSTFastWith("/api/approvals/:id/approve", approvalController.Approve, jwtMiddleware)
	router.POSTFastWith("/api/approvals/:id/reject", approvalController.Reject, jwtMiddleware)

	// Public health check (no auth required)
	router.GETFast("/api/health", func(c *web.FastRequestContext) error {
		health := map[string]interface{}{
			"status":    "ok",
			"timestamp": time.Now().Unix(),
		}

		if db != nil {
			stats := db.Stats()
			health["database"] = map[string]interface{}{
				"connected": stats.OpenConnections > 0,
				"in_use":    stats.InUse,
				"idle":      stats.Idle,
			}
		}

		return c.JSON(200, health)
	})

	return server, nil
}

// serverMetricsAdapter adapts *web.FastHTTPServer to dashboard.HTTPServerMetricsProvider.
// web.ServerMetrics and dashboard.HTTPServerMetricsData have identical fields.
type serverMetricsAdapter struct{ s *web.FastHTTPServer }

func (a *serverMetricsAdapter) Metrics() dashboard.HTTPServerMetricsData {
	m := a.s.Metrics()
	return dashboard.HTTPServerMetricsData{
		QueuedRequests:       m.QueuedRequests,
		RejectedRequests:     m.RejectedRequests,
		TotalRequests:        m.TotalRequests,
		SuccessfulRequests:   m.SuccessfulRequests,
		ErrorRequests:        m.ErrorRequests,
		QueueCapacity:        m.QueueCapacity,
		QueueUtilization:     m.QueueUtilization,
		Workers:              m.Workers,
		CurrentCCU:           m.CurrentCCU,
		CCUUtilization:       m.CCUUtilization,
		BytesSent:            m.BytesSent,
		BytesReceived:        m.BytesReceived,
		AverageLatencyMs:     m.AverageLatencyMs,
		ArrivalRate:          m.ArrivalRate,
		ExpectedQueueLength:  m.ExpectedQueueLength,
		LittlesLawValidation: m.LittlesLawValidation,
	}
}
