// Copyright (c) 2026 Alexander G.
// Author: Alexander G. (Samsonix)
// License: MIT
// Project: EyesOn SIM Management System

package handlers

import (
	"fmt"
	"time"

	"eyeson-go-server/internal/config"
	"eyeson-go-server/internal/database"
	"eyeson-go-server/internal/models"
	"eyeson-go-server/internal/services"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	var user models.User
	result := database.DB.Preload("Role").Where("username = ?", req.Username).First(&user)
	if result.Error != nil {
		// Log failed login attempt (user not found)
		services.Audit.LogLogin(c, 0, req.Username, "", false, "User not found")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid credentials"})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		// Log failed login attempt (wrong password)
		services.Audit.LogLogin(c, user.ID, user.Username, user.Role.Name, false, "Invalid password")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid credentials"})
	}

	// Check if user is active
	if !user.IsActive {
		services.Audit.LogLogin(c, user.ID, user.Username, user.Role.Name, false, "Account disabled")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Account is disabled"})
	}

	cfg, _ := config.LoadConfig()
	sessionID := uuid.New().String()

	claims := jwt.MapClaims{
		"user_id":    user.ID,
		"username":   user.Username,
		"role":       user.Role.Name,
		"session_id": sessionID,
		"exp":        time.Now().Add(time.Hour * 24).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	t, err := token.SignedString([]byte(cfg.JwtSecret))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not login"})
	}

	// Update last seen
	database.DB.Model(&user).Update("last_seen", time.Now())

	// Log successful login
	services.Audit.LogLogin(c, user.ID, user.Username, user.Role.Name, true, "")

	return c.JSON(fiber.Map{
		"token":    t,
		"user_id":  user.ID,
		"username": user.Username,
		"role":     user.Role.Name,
	})
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

func ChangePassword(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	var req ChangePasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	if req.NewPassword == "" || len(req.NewPassword) < 6 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "New password must be at least 6 characters"})
	}

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.OldPassword)); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid old password"})
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not hash password"})
	}

	user.PasswordHash = string(hashedPassword)
	if err := database.DB.Save(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not update password"})
	}

	return c.JSON(fiber.Map{"message": "Password changed successfully"})
}

func GetUsers(c *fiber.Ctx) error {
	var users []models.User
	if err := database.DB.Preload("Role").Find(&users).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not fetch users"})
	}

	type UserResponse struct {
		ID        uint      `json:"id"`
		Username  string    `json:"username"`
		Email     string    `json:"email"`
		Role      string    `json:"role"`
		IsActive  bool      `json:"is_active"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
	}

	var response []UserResponse
	for _, user := range users {
		response = append(response, UserResponse{
			ID:        user.ID,
			Username:  user.Username,
			Email:     user.Email,
			Role:      user.Role.Name,
			IsActive:  user.IsActive,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
		})
	}

	return c.JSON(fiber.Map{"data": response})
}

type CreateUserRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

func CreateUser(c *fiber.Ctx) error {
	var req CreateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	if req.Username == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Username and password are required"})
	}

	if len(req.Password) < 6 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Password must be at least 6 characters"})
	}

	// Find role by name
	var role models.Role
	if req.Role != "" {
		if err := database.DB.Where("name = ?", req.Role).First(&role).Error; err != nil {
			// Default to Viewer role
			database.DB.Where("name = ?", "Viewer").First(&role)
		}
	} else {
		database.DB.Where("name = ?", "Viewer").First(&role)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not hash password"})
	}

	user := models.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		RoleID:       role.ID,
		IsActive:     true,
	}

	if err := database.DB.Create(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not create user"})
	}

	// Log user creation
	services.Audit.LogUserCreate(c, user.ID, user.Username)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "User created successfully",
		"user_id": user.ID,
	})
}

type UpdateUserRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	IsActive *bool  `json:"is_active"`
}

func UpdateUser(c *fiber.Ctx) error {
	userID := c.Params("id")

	var req UpdateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	var user models.User
	if err := database.DB.Preload("Role").First(&user, userID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}

	// Track changes for audit
	oldUsername := user.Username
	oldEmail := user.Email
	oldRole := user.Role.Name
	oldIsActive := user.IsActive

	if req.Username != "" && req.Username != user.Username {
		user.Username = req.Username
		services.Audit.LogUserUpdate(c, user.ID, "username", oldUsername, req.Username)
	}

	if req.Email != "" && req.Email != user.Email {
		user.Email = req.Email
		services.Audit.LogUserUpdate(c, user.ID, "email", oldEmail, req.Email)
	}

	if req.Role != "" && req.Role != user.Role.Name {
		var role models.Role
		if err := database.DB.Where("name = ?", req.Role).First(&role).Error; err == nil {
			// Update both the ID for saving and the in-memory struct for the response
			user.RoleID = role.ID
			user.Role = role
			services.Audit.LogUserUpdate(c, user.ID, "role", oldRole, role.Name)
		}
	}

	if req.IsActive != nil && *req.IsActive != oldIsActive {
		user.IsActive = *req.IsActive
		services.Audit.LogUserUpdate(c, user.ID, "is_active", fmt.Sprintf("%v", oldIsActive), fmt.Sprintf("%v", *req.IsActive))
	}

	if err := database.DB.Save(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not update user"})
	}

	// Reload user with updated role to return correct data
	database.DB.Preload("Role").First(&user, userID)

	// Format user data consistently with GetUsers
	responseUser := fiber.Map{
		"id":         user.ID,
		"username":   user.Username,
		"email":      user.Email,
		"role":       user.Role.Name,
		"is_active":  user.IsActive,
		"created_at": user.CreatedAt,
		"updated_at": user.UpdatedAt,
	}

	return c.JSON(fiber.Map{
		"message": "User updated successfully",
		"user":    responseUser,
	})
}

func DeleteUser(c *fiber.Ctx) error {
	userID := c.Params("id")

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}

	username := user.Username

	if err := database.DB.Delete(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not delete user"})
	}

	// Log user deletion
	services.Audit.LogUserDelete(c, user.ID, username)

	return c.JSON(fiber.Map{"message": "User deleted successfully"})
}

type ResetPasswordRequest struct {
	NewPassword string `json:"new_password"`
}

func ResetUserPassword(c *fiber.Ctx) error {
	userID := c.Params("id")

	var req ResetPasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	if req.NewPassword == "" || len(req.NewPassword) < 6 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "New password must be at least 6 characters"})
	}

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not hash password"})
	}

	user.PasswordHash = string(hashedPassword)
	if err := database.DB.Save(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not reset password"})
	}

	return c.JSON(fiber.Map{"message": "Password reset successfully"})
}
