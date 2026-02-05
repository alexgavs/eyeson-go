// Copyright (c) 2026 Alexander G.
// Author: Alexander G. (Samsonix)
// License: MIT
// Project: EyesOn SIM Management System

package handlers

import (
	"fmt"
	"strings"

	"eyeson-go-server/internal/config"
	"eyeson-go-server/internal/database"
	"eyeson-go-server/internal/models"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

func JWTMiddleware(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Missing authorization header"})
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == authHeader {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid authorization format"})
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Server configuration error"})
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Pin signing method to HS256.
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %s", token.Header["alg"])
		}
		return []byte(cfg.JwtSecret), nil
	})

	if err != nil || !token.Valid {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid or expired token"})
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token claims"})
	}

	userID, err := claimUint(claims, "user_id")
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token claims"})
	}
	username, err := claimString(claims, "username")
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token claims"})
	}
	role, _ := claimString(claims, "role")
	sessionID, _ := claimString(claims, "session_id")

	// Canonical locals for handlers
	c.Locals("user_id", userID)
	c.Locals("username", username)
	c.Locals("role", role)
	c.Locals("session_id", sessionID)

	// Backward/forward compat: audit service expects Locals("user") to contain jwt.MapClaims
	c.Locals("user", claims)

	return c.Next()
}

func claimString(claims jwt.MapClaims, key string) (string, error) {
	v, ok := claims[key]
	if !ok {
		return "", fmt.Errorf("missing claim %s", key)
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("claim %s is not string", key)
	}
	return s, nil
}

func claimUint(claims jwt.MapClaims, key string) (uint, error) {
	v, ok := claims[key]
	if !ok {
		return 0, fmt.Errorf("missing claim %s", key)
	}
	// jwt.MapClaims comes from JSON decoding; numbers are float64.
	if f, ok := v.(float64); ok {
		if f < 0 {
			return 0, fmt.Errorf("claim %s negative", key)
		}
		return uint(f), nil
	}
	if i, ok := v.(int); ok {
		if i < 0 {
			return 0, fmt.Errorf("claim %s negative", key)
		}
		return uint(i), nil
	}
	if u, ok := v.(uint); ok {
		return u, nil
	}
	return 0, fmt.Errorf("claim %s has unsupported type", key)
}

// RequireRole middleware checks if user has a specific role
func RequireRole(requiredRole string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := c.Locals("user_id").(uint)

		var user models.User
		if err := database.DB.Preload("Role").First(&user, userID).Error; err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "User not found"})
		}

		if user.Role.Name != requiredRole {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Access denied: insufficient permissions",
			})
		}

		return c.Next()
	}
}

// RequireAnyRole middleware checks if user has any of the specified roles
func RequireAnyRole(allowedRoles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := c.Locals("user_id").(uint)

		var user models.User
		if err := database.DB.Preload("Role").First(&user, userID).Error; err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "User not found"})
		}

		for _, role := range allowedRoles {
			if user.Role.Name == role {
				c.Locals("role", user.Role.Name)
				return c.Next()
			}
		}

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied: insufficient permissions",
		})
	}
}
