// Copyright (c) 2026 Alexander G.
// Author: Alexander G. (Samsonix)
// License: MIT
// Project: EyesOn SIM Management System

package routes

import (
	"strings"

	"eyeson-go-server/internal/config"
	"eyeson-go-server/internal/handlers"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

func SetupRoutes(app *fiber.App, cfg *config.Config) {
	app.Use(recover.New())
	app.Use(requestid.New())

	allowOrigins := strings.TrimSpace(cfg.CorsAllowOrigins)
	if allowOrigins == "" {
		allowOrigins = "http://localhost:5000,http://127.0.0.1:5000"
	}

	app.Use(cors.New(cors.Config{
		AllowOrigins:     allowOrigins,
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowCredentials: false,
	}))
	app.Use(logger.New())

	// Initialize Google OAuth
	handlers.InitGoogleOAuth(cfg)

	// API routes
	api := app.Group("/api/v1")

	// Auth routes (public)
	api.Post("/auth/login", handlers.Login)

	// Google OAuth routes (public)
	api.Get("/auth/google/config", handlers.GetGoogleOAuthConfig)
	api.Get("/auth/google/login", handlers.GoogleLogin)
	api.Get("/auth/google/callback", handlers.GoogleCallback)

	// Auth routes (protected)
	authProtected := api.Group("/auth")
	authProtected.Use(handlers.JWTMiddleware)
	authProtected.Put("/change-password", handlers.ChangePassword)
	// Google account management
	authProtected.Get("/google/status", handlers.GetUserGoogleStatus)
	authProtected.Post("/google/link", handlers.LinkGoogleAccount)
	authProtected.Delete("/google/unlink", handlers.UnlinkGoogleAccount)
	authProtected.Get("/google/link/callback", handlers.GoogleLinkCallback) // Moved here to be behind JWT middleware

	// User management routes (protected - Admin only)
	users := api.Group("/users")
	users.Use(handlers.JWTMiddleware)
	users.Use(handlers.RequireAnyRole("Administrator"))
	users.Get("/", handlers.GetUsers)
	users.Post("/", handlers.CreateUser)
	users.Put("/:id", handlers.UpdateUser)
	users.Delete("/:id", handlers.DeleteUser)
	users.Post("/:id/reset-password", handlers.ResetUserPassword)

	// Role management routes (protected - Admin only)
	roles := api.Group("/roles")
	roles.Use(handlers.JWTMiddleware)
	roles.Use(handlers.RequireRole("Administrator"))
	roles.Get("/", handlers.GetRoles)
	roles.Get("/:id", handlers.GetRole)
	roles.Post("/", handlers.CreateRole)
	roles.Put("/:id", handlers.UpdateRole)
	roles.Delete("/:id", handlers.DeleteRole)

	// SIM routes (protected - All roles can read, Admin+Moderator can write)
	sims := api.Group("/sims")
	sims.Use(handlers.JWTMiddleware)
	sims.Get("/", handlers.GetSims) // All authenticated users
	sims.Get("/:msisdn/history", handlers.GetSimHistory)

	simsWrite := sims.Group("")
	simsWrite.Use(handlers.RequireAnyRole("Administrator", "Moderator"))
	simsWrite.Post("/update", handlers.UpdateSim)
	simsWrite.Post("/status", handlers.ChangeStatus) // Single SIM status change with queue fallback
	simsWrite.Post("/bulk-status", handlers.BulkChangeStatus)

	// Stats routes (protected - All roles)
	stats := api.Group("/stats")
	stats.Use(handlers.JWTMiddleware)
	stats.Get("/", handlers.GetStats)

	// API Status route (Admin only - shows API tokens and connection info)
	apiStatus := api.Group("/api-status")
	apiStatus.Use(handlers.JWTMiddleware)
	apiStatus.Use(handlers.RequireRole("Administrator"))
	apiStatus.Get("/", handlers.GetAPIStatus)
	apiStatus.Get("/diagnostics", handlers.GetAPIDiagnostics)

	// API Connection Toggle (Admin only)
	api.Post("/api-connection", handlers.JWTMiddleware, handlers.RequireRole("Administrator"), handlers.ToggleConnection)

	// Upstream selection (Admin only) - persisted, applies after restart
	upstream := api.Group("/upstream")
	upstream.Use(handlers.JWTMiddleware)
	upstream.Use(handlers.RequireRole("Administrator"))
	upstream.Get("", handlers.GetUpstream)
	upstream.Put("", handlers.SetUpstream)

	// Sync status - available to all authenticated users
	api.Get("/sync/status", handlers.JWTMiddleware, handlers.GetManualSyncStatus)

	// Manual sync trigger (Admin only) - pulls latest data from Pelephone into local DB
	syncAdmin := api.Group("/sync")
	syncAdmin.Use(handlers.JWTMiddleware)
	syncAdmin.Use(handlers.RequireRole("Administrator"))
	syncAdmin.Post("/full", handlers.TriggerManualFullSync)

	// Jobs routes (protected - All roles)
	jobs := api.Group("/jobs")
	jobs.Use(handlers.JWTMiddleware)
	jobs.Get("/", handlers.GetJobs)
	jobs.Get("/queue", handlers.GetSyncTasks)
	jobs.Post("/queue/:id/execute", handlers.ExecuteQueueTask) // Instant execution
	jobs.Get("/local/:id", handlers.GetLocalJob)

	// Queue management routes (protected)
	queue := api.Group("/queue")
	queue.Use(handlers.JWTMiddleware)
	// User's own queue operations
	queue.Get("/my", handlers.GetMyQueue)                             // Current user's active tasks
	queue.Get("/my/history", handlers.GetMyQueueHistory)              // User's completed tasks history
	queue.Get("/task/:id", handlers.GetTaskStatus)                    // Get task by ID
	queue.Get("/request/:request_id", handlers.GetTaskByRequestID)    // Get task by request ID
	queue.Get("/batch/:batch_id", handlers.GetBatchTasks)             // Get all tasks in batch
	queue.Get("/batch/:batch_id/progress", handlers.GetBatchProgress) // Get batch progress
	queue.Post("/task/:id/cancel", handlers.CancelTask)               // Cancel own task

	// Admin queue operations
	queueAdmin := queue.Group("")
	queueAdmin.Use(handlers.RequireAnyRole("Administrator"))
	queueAdmin.Get("/all", handlers.GetAllPendingTasks)                 // All pending tasks
	queueAdmin.Get("/stats", handlers.GetQueueStats)                    // Queue statistics
	queueAdmin.Post("/task/:id/cancel-admin", handlers.CancelTaskAdmin) // Admin cancel any task
	queueAdmin.Post("/task/:id/retry", handlers.RetryTask)              // Retry failed task
	queueAdmin.Delete("/cleanup", handlers.CleanupOldTasks)             // Cleanup old completed tasks

	// Audit log routes (protected - Admin only)
	audit := api.Group("/audit")
	audit.Use(handlers.JWTMiddleware)
	audit.Use(handlers.RequireAnyRole("Administrator"))
	audit.Get("/", handlers.GetAuditLogs)                     // List audit logs with filtering
	audit.Get("/stats", handlers.GetAuditStats)               // Audit statistics
	audit.Get("/entity/:type/:id", handlers.GetEntityHistory) // Entity history
	audit.Get("/sim/:msisdn", handlers.GetSIMHistory)         // SIM-specific history
	audit.Get("/user/:id/activity", handlers.GetUserActivity) // User activity
	audit.Get("/export", handlers.ExportAuditLogs)            // Export to CSV
	audit.Delete("/cleanup", handlers.CleanupOldAuditLogs)    // Cleanup old logs

	// Reactive endpoints (protected)
	reactive := api.Group("/reactive")
	reactive.Use(handlers.JWTMiddleware)
	reactive.Get("/events", handlers.ReactiveEventsHandler)    // Reactive SSE event stream
	reactive.Get("/sims", handlers.ReactiveSimsListHandler)    // Reactive SIM list
	reactive.Get("/search", handlers.ReactiveSimSearchHandler) // Reactive debounced search
	reactive.Get("/stats", handlers.ReactiveStatsHandler)      // Reactive aggregated stats

	// Статические файлы
	app.Static("/assets", "./static/assets")
	app.Static("/", "./static")

	// Swagger documentation (dev-only by config)
	if cfg.EnableSwagger {
		app.Get("/api/docs", func(c *fiber.Ctx) error {
			return c.Redirect("/swagger.html")
		})
		app.Get("/docs", func(c *fiber.Ctx) error {
			return c.Redirect("/swagger.html")
		})
	}

	// Redirect root to React dashboard
	app.Get("/", func(c *fiber.Ctx) error {
		return c.Redirect("/index.html")
	})
}
