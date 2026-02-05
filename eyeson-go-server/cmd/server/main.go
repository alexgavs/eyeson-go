// Copyright (c) 2026 Alexander G.
// Author: Alexander G. (Samsonix)
// License: MIT
// Project: EyesOn SIM Management System

package main

import (
	"log"

	"eyeson-go-server/internal/config"
	"eyeson-go-server/internal/database"
	"eyeson-go-server/internal/eyesont"
	"eyeson-go-server/internal/jobs"
	"eyeson-go-server/internal/routes"
	"eyeson-go-server/internal/services"
	"eyeson-go-server/internal/syncer"

	"github.com/gofiber/fiber/v2"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Could not load config: %v", err)
	}

	// Connect to local database (DB-first architecture)
	database.Connect(cfg)

	// Resolve upstream selection (persisted in DB). Switching applies after restart.
	selectedUpstream, err := services.GetUpstreamSelected()
	if err != nil {
		log.Printf("[Upstream] WARNING: could not read selection, defaulting to pelephone: %v", err)
		selectedUpstream = services.UpstreamPelephone
	}
	resolvedBaseURL := services.ResolveUpstreamBaseURL(cfg, selectedUpstream)
	log.Printf("[Upstream] Selected=%s BaseURL=%s", selectedUpstream, resolvedBaseURL)

	// Initialize EyesOnT API client (connects to external Pelephone server or simulator)
	eyesont.InitWithConfig(resolvedBaseURL, cfg.ApiUsername, cfg.ApiPassword, cfg.ApiDelayMs, cfg.ApiInsecureTLS)

	// Start background sync service (synchronizes data from API to local DB)
	syncService := syncer.New(database.DB)
	syncService.Start()

	// Start job worker (processes queued tasks)
	jobWorker := jobs.New(database.DB)
	jobWorker.Start()

	// Create and configure Fiber app
	app := fiber.New()
	routes.SetupRoutes(app, cfg)

	// Start HTTP server
	log.Printf("Server starting on port %s", cfg.Port)
	if err := app.Listen(":" + cfg.Port); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
