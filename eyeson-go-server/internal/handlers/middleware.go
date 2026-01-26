package handlers

import (
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

	cfg, _ := config.LoadConfig()
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(cfg.JwtSecret), nil
	})

	if err != nil || !token.Valid {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid or expired token"})
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token claims"})
	}

	userID := uint(claims["user_id"].(float64))
	username := claims["username"].(string)

	c.Locals("user_id", userID)
	c.Locals("username", username)

	return c.Next()
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

// RequirePermission middleware checks if user has a specific permission
func RequirePermission(permission string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := c.Locals("user_id").(uint)

		var user models.User
		if err := database.DB.Preload("Role").First(&user, userID).Error; err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "User not found"})
		}

		// Check if role has the required permission
		if !strings.Contains(user.Role.Permissions, permission) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Access denied: missing required permission",
			})
		}

		return c.Next()
	}
}
