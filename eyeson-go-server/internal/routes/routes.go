package routes

import (
	"eyeson-go-server/internal/handlers"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func SetupRoutes(app *fiber.App) {
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "*",
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowCredentials: false,
	}))
	app.Use(logger.New())

	// API routes
	api := app.Group("/api/v1")

	// Auth routes (public)
	api.Post("/auth/login", handlers.Login)

	// Auth routes (protected)
	authProtected := api.Group("/auth")
	authProtected.Use(handlers.JWTMiddleware)
	authProtected.Put("/change-password", handlers.ChangePassword)

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

	// API Connection Toggle (Admin only)
	api.Post("/api-connection", handlers.JWTMiddleware, handlers.RequireRole("Administrator"), handlers.ToggleConnection)

	// Jobs routes (protected - All roles)
	jobs := api.Group("/jobs")
	jobs.Use(handlers.JWTMiddleware)
	jobs.Get("/", handlers.GetJobs)
	jobs.Get("/queue", handlers.GetSyncTasks)
	jobs.Post("/queue/:id/execute", handlers.ExecuteQueueTask) // Instant execution
	jobs.Get("/local/:id", handlers.GetLocalJob)

	// Статические файлы
	app.Static("/assets", "./static/assets")
	app.Static("/", "./static")

	// Swagger documentation
	app.Get("/api/docs", func(c *fiber.Ctx) error {
		return c.Redirect("/swagger.html")
	})
	app.Get("/docs", func(c *fiber.Ctx) error {
		return c.Redirect("/swagger.html")
	})

	// Redirect root to React dashboard
	app.Get("/", func(c *fiber.Ctx) error {
		return c.Redirect("/index.html")
	})
}
