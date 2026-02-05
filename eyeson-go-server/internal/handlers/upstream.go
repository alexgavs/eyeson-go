// Copyright (c) 2026 Alexander G.
// Author: Alexander G. (Samsonix)
// License: MIT
// Project: EyesOn SIM Management System

package handlers

import (
	"eyeson-go-server/internal/config"
	"eyeson-go-server/internal/services"

	"github.com/gofiber/fiber/v2"
)

type upstreamSetRequest struct {
	Selected string `json:"selected"`
}

// GetUpstream returns current upstream selection and both configured options.
// GET /api/v1/upstream (Admin only)
func GetUpstream(c *fiber.Ctx) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Could not load config: " + err.Error()})
	}

	selected, selErr := services.GetUpstreamSelected()
	if selErr != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Could not read upstream selection: " + selErr.Error()})
	}

	return c.JSON(fiber.Map{
		"selected": string(selected),
		"options": fiber.Map{
			"pelephone": fiber.Map{
				"base_url": cfg.ApiBaseUrl,
			},
			"simulator": fiber.Map{
				"base_url": cfg.SimulatorBaseUrl,
			},
		},
		"restart_required": false,
	})
}

// SetUpstream persists upstream selection. Applies after server restart.
// PUT /api/v1/upstream (Admin only)
func SetUpstream(c *fiber.Ctx) error {
	var req upstreamSetRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}

	selected, ok := services.NormalizeUpstreamSelection(req.Selected)
	if !ok {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid selected value (expected pelephone|simulator)"})
	}

	if err := services.SetUpstreamSelected(selected); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to save upstream selection: " + err.Error()})
	}

	cfg, _ := config.LoadConfig()
	return c.JSON(fiber.Map{
		"selected": string(selected),
		"options": fiber.Map{
			"pelephone": fiber.Map{
				"base_url": cfg.ApiBaseUrl,
			},
			"simulator": fiber.Map{
				"base_url": cfg.SimulatorBaseUrl,
			},
		},
		"restart_required": true,
	})
}
