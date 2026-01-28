// Copyright (c) 2026 Alexander G.
// Author: Alexander G. (Samsonix)
// License: MIT
// Project: EyesOn SIM Management System

package handlers

import (
	"eyeson-go-server/internal/database"
	"eyeson-go-server/internal/models"

	"github.com/gofiber/fiber/v2"
)

type CreateRoleRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Permissions string `json:"permissions"`
}

type UpdateRoleRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Permissions string `json:"permissions"`
}

func GetRoles(c *fiber.Ctx) error {
	var roles []models.Role
	if err := database.DB.Find(&roles).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not fetch roles"})
	}

	return c.JSON(fiber.Map{"data": roles})
}

func GetRole(c *fiber.Ctx) error {
	roleID := c.Params("id")

	var role models.Role
	if err := database.DB.First(&role, roleID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Role not found"})
	}

	return c.JSON(role)
}

func CreateRole(c *fiber.Ctx) error {
	var req CreateRoleRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Role name is required"})
	}

	role := models.Role{
		Name:        req.Name,
		Description: req.Description,
		Permissions: req.Permissions,
	}

	if err := database.DB.Create(&role).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not create role"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Role created successfully",
		"role_id": role.ID,
	})
}

func UpdateRole(c *fiber.Ctx) error {
	roleID := c.Params("id")

	var req UpdateRoleRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	var role models.Role
	if err := database.DB.First(&role, roleID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Role not found"})
	}

	if req.Name != "" {
		role.Name = req.Name
	}

	if req.Description != "" {
		role.Description = req.Description
	}

	if req.Permissions != "" {
		role.Permissions = req.Permissions
	}

	if err := database.DB.Save(&role).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not update role"})
	}

	return c.JSON(fiber.Map{"message": "Role updated successfully"})
}

func DeleteRole(c *fiber.Ctx) error {
	roleID := c.Params("id")

	var role models.Role
	if err := database.DB.First(&role, roleID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Role not found"})
	}

	// Check if any users have this role
	var userCount int64
	database.DB.Model(&models.User{}).Where("role_id = ?", roleID).Count(&userCount)
	if userCount > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot delete role: users are still assigned to it",
			"count": userCount,
		})
	}

	if err := database.DB.Delete(&role).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not delete role"})
	}

	return c.JSON(fiber.Map{"message": "Role deleted successfully"})
}
