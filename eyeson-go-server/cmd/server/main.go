package main

import (
	"log"
	"os"
	"path/filepath"

	"eyeson-go-server/internal/config"
	"eyeson-go-server/internal/database"
	"eyeson-go-server/internal/eyesont"
	"eyeson-go-server/internal/routes"

	"github.com/gofiber/fiber/v2"
)

func main() {
	// Get executable directory
	exePath, err := os.Executable()
	if err != nil {
		log.Fatalf("Could not get executable path: %v", err)
	}
	exeDir := filepath.Dir(exePath)

	// Change working directory to executable location
	if err := os.Chdir(exeDir); err != nil {
		log.Fatalf("Could not change directory: %v", err)
	}

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
