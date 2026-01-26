package main

import (
	"log"

	"eyeson-go-server/internal/config"
	"eyeson-go-server/internal/database"
	"eyeson-go-server/internal/eyesont"
	"eyeson-go-server/internal/routes"

	"github.com/gofiber/fiber/v2"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Could not load config: %v", err)
	}

	database.Connect(cfg)

	eyesont.InitWithConfig(cfg.ApiBaseUrl, cfg.ApiUsername, cfg.ApiPassword)

	app := fiber.New()

	app.Static("/", "./static")

	routes.SetupRoutes(app)

	log.Printf("Server starting on port %s", cfg.Port)
	if err := app.Listen(":" + cfg.Port); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
