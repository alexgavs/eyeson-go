// Copyright (c) 2026 Alexander G.
// Author: Alexander G. (Samsonix)
// License: MIT
// Project: EyesOn SIM Management System

package handlers

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"eyeson-go-server/internal/config"
	"eyeson-go-server/internal/database"
	"eyeson-go-server/internal/models"
	"eyeson-go-server/internal/services"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var googleOAuthConfig *oauth2.Config
var Audit = services.Audit

// InitGoogleOAuth initializes Google OAuth configuration
func InitGoogleOAuth(cfg *config.Config) {
	if !cfg.GoogleEnabled || cfg.GoogleClientID == "" {
		return
	}

	googleOAuthConfig = &oauth2.Config{
		ClientID:     cfg.GoogleClientID,
		ClientSecret: cfg.GoogleClientSecret,
		RedirectURL:  cfg.GoogleRedirectURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}
}

// GoogleUserInfo represents Google user info response
type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
}

// GetGoogleOAuthConfig returns current OAuth config status
func GetGoogleOAuthConfig(c *fiber.Ctx) error {
	cfg, _ := config.LoadConfig()

	return c.JSON(fiber.Map{
		"enabled":   cfg.GoogleEnabled,
		"client_id": cfg.GoogleClientID, // Safe to expose client ID
	})
}

// GoogleLogin initiates Google OAuth flow
func GoogleLogin(c *fiber.Ctx) error {
	cfg, _ := config.LoadConfig()

	if !cfg.GoogleEnabled || googleOAuthConfig == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Google OAuth is not configured",
		})
	}

	// Generate state token for CSRF protection
	state := uuid.New().String()

	isSecure := c.Secure()
	if strings.HasPrefix(c.Hostname(), "localhost") {
		isSecure = false
	}

	// Store state in session/cookie for verification
	c.Cookie(&fiber.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Expires:  time.Now().Add(10 * time.Minute),
		HTTPOnly: true,
		Secure:   isSecure,
		SameSite: "Lax",
		Path:     "/", // Explicitly set path to root
	})

	url := googleOAuthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.SetAuthURLParam("prompt", "consent"))
	return c.Redirect(url)
}

// GoogleCallback handles Google OAuth callback
func GoogleCallback(c *fiber.Ctx) error {
	cfg, _ := config.LoadConfig()

	if !cfg.GoogleEnabled || googleOAuthConfig == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Google OAuth is not configured",
		})
	}

	// Verify state
	state := c.Query("state")
	savedState := c.Cookies("oauth_state")

	// --- Enhanced Debug Logging ---
	log.Println("--- Google OAuth Callback Debug ---")
	log.Printf("State from URL: %s", state)
	log.Printf("State from 'oauth_state' cookie: %s", savedState)
	c.Request().Header.VisitAllCookie(func(key, value []byte) {
		log.Printf("Found cookie: %s = %s", string(key), string(value))
	})
	log.Println("---------------------------------")
	// --- End Debug Logging ---

	if state == "" || savedState == "" || state != savedState {
		log.Println("[OAuth] Error: Invalid OAuth state")
		return c.Redirect("/?error=" + url.QueryEscape("Invalid OAuth state"))
	}

	// Clear state cookie immediately after use
	c.Cookie(&fiber.Cookie{
		Name:     "oauth_state",
		Value:    "",
		Expires:  time.Now().Add(-1 * time.Hour),
		HTTPOnly: true,
		Secure:   c.Secure(),
		SameSite: "Lax",
	})

	// Exchange code for token
	code := c.Query("code")
	if code == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Missing authorization code",
		})
	}

	token, err := googleOAuthConfig.Exchange(context.Background(), code)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to exchange token: " + err.Error(),
		})
	}

	// Get user info from Google
	userInfo, err := getGoogleUserInfo(token.AccessToken)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get user info: " + err.Error(),
		})
	}

	// Find or create user
	var user models.User
	result := database.DB.Preload("Role").Where("google_id = ?", userInfo.ID).First(&user)

	if result.Error != nil {
		// User not found by Google ID, try to find by email
		result = database.DB.Preload("Role").Where("email = ?", userInfo.Email).First(&user)

		if result.Error != nil {
			// Create new user
			var viewerRole models.Role
			database.DB.Where("name = ?", "Viewer").First(&viewerRole)

			// Generate random password for Google users
			randomPass := uuid.New().String()
			hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(randomPass), bcrypt.DefaultCost)

			user = models.User{
				Username:     userInfo.Email,
				Email:        userInfo.Email,
				PasswordHash: string(hashedPassword),
				GoogleID:     userInfo.ID,
				AvatarURL:    userInfo.Picture,
				RoleID:       viewerRole.ID,
				IsActive:     true,
			}

			if err := database.DB.Create(&user).Error; err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Failed to create user: " + err.Error(),
				})
			}

			// Reload with role
			database.DB.Preload("Role").First(&user, user.ID)

			services.Audit.LogUserCreate(c, user.ID, user.Username)

		} else {
			// User found by email - link Google account
			user.GoogleID = userInfo.ID
			user.AvatarURL = userInfo.Picture
			database.DB.Save(&user)
		}
	} else {
		// Update avatar if changed
		if user.AvatarURL != userInfo.Picture {
			user.AvatarURL = userInfo.Picture
			database.DB.Save(&user)
		}
	}

	// Check if user is active
	if !user.IsActive {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Account is disabled",
		})
	}

	// Generate JWT token
	sessionID := uuid.New().String()
	claims := jwt.MapClaims{
		"user_id":    user.ID,
		"username":   user.Username,
		"role":       user.Role.Name,
		"session_id": sessionID,
		"exp":        time.Now().Add(time.Hour * 24).Unix(),
	}
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	t, err := jwtToken.SignedString([]byte(cfg.JwtSecret))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not generate token",
		})
	}

	// Update last seen
	database.DB.Model(&user).Update("last_seen", time.Now())

	// Log successful login
	services.Audit.LogLogin(c, user.ID, user.Username, user.Role.Name, true, "Google OAuth")

	// Redirect to frontend with token
	redirectURL := "/?token=" + t + "&user=" + user.Username + "&role=" + user.Role.Name
	return c.Redirect(redirectURL)
}

// LinkGoogleAccount links Google account to existing user
func LinkGoogleAccount(c *fiber.Ctx) error {
	cfg, _ := config.LoadConfig()

	if !cfg.GoogleEnabled || googleOAuthConfig == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Google OAuth is not configured",
		})
	}

	// Generate state and store in cookie
	state := uuid.New().String()
	isSecure := c.Secure()
	if strings.HasPrefix(c.Hostname(), "localhost") {
		isSecure = false
	}
	c.Cookie(&fiber.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Expires:  time.Now().Add(10 * time.Minute),
		HTTPOnly: true,
		Secure:   isSecure,
		SameSite: "Lax",
		Path:     "/", // Explicitly set path to root
	})

	// Redirect URL will be slightly different to indicate a 'link' action
	linkRedirectURL := strings.Replace(cfg.GoogleRedirectURL, "/callback", "/link/callback", 1)
	linkOAuthConfig := *googleOAuthConfig // Create a copy
	linkOAuthConfig.RedirectURL = linkRedirectURL

	url := linkOAuthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.SetAuthURLParam("prompt", "consent"))
	return c.JSON(fiber.Map{
		"redirect_url": url,
	})
}

// GoogleLinkCallback handles callback for account linking
func GoogleLinkCallback(c *fiber.Ctx) error {
	cfg, _ := config.LoadConfig()

	if !cfg.GoogleEnabled || googleOAuthConfig == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Google OAuth is not configured",
		})
	}

	// Verify state from the same cookie as login
	state := c.Query("state")
	savedState := c.Cookies("oauth_state")
	if state == "" || savedState == "" || state != savedState {
		log.Println("[OAuth Link] Error: Invalid OAuth state")
		return c.Redirect("/profile?link_error=" + url.QueryEscape("Invalid state token"))
	}

	// Clear state cookie
	c.Cookie(&fiber.Cookie{
		Name:     "oauth_state",
		Value:    "",
		Expires:  time.Now().Add(-1 * time.Hour),
		HTTPOnly: true,
		Secure:   c.Secure(),
		SameSite: "Lax",
		Path:     "/",
	})

	// Get user from JWT - this is now safe because the route is protected
	tokenData := c.Locals("user").(*jwt.Token)
	claims := tokenData.Claims.(jwt.MapClaims)
	userID := uint(claims["user_id"].(float64))

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return c.Redirect("/profile?link_error=" + url.QueryEscape("User not found"))
	}

	// Exchange code for token
	code := c.Query("code")
	if code == "" {
		return c.Redirect("/profile?link_error=" + url.QueryEscape("Missing authorization code"))
	}

	linkRedirectURL := strings.Replace(cfg.GoogleRedirectURL, "/callback", "/link/callback", 1)
	linkOAuthConfig := *googleOAuthConfig // Create a copy
	linkOAuthConfig.RedirectURL = linkRedirectURL

	token, err := linkOAuthConfig.Exchange(context.Background(), code)
	if err != nil {
		return c.Redirect("/profile?link_error=" + url.QueryEscape("Failed to exchange token: "+err.Error()))
	}

	// Get user info
	userInfo, err := getGoogleUserInfo(token.AccessToken)
	if err != nil {
		return c.Redirect("/profile?link_error=" + url.QueryEscape("Failed to get user info: "+err.Error()))
	}

	// Check if this Google account is already linked to another user
	var existingUser models.User
	if database.DB.Where("google_id = ? AND id != ?", userInfo.ID, user.ID).First(&existingUser).Error == nil {
		return c.Redirect("/profile?link_error=" + url.QueryEscape("Google account already linked to another user"))
	}

	// Link account
	user.GoogleID = userInfo.ID
	user.AvatarURL = userInfo.Picture
	if err := database.DB.Save(&user).Error; err != nil {
		return c.Redirect("/profile?link_error=" + url.QueryEscape("Failed to update user"))
	}

	// Correctly log the linking action
	auditLog := services.Audit.NewLog(c)
	auditLog.SetAction(models.ActionGoogleLink).
		SetStatus(models.AuditStatusSuccess).
		SetDetails("User linked Google account successfully.")
	auditLog.Save()

	return c.Redirect("/profile?link_success=true")
}

// UnlinkGoogleAccount removes Google account link
func UnlinkGoogleAccount(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	// Make sure user has a password set (can still login)
	if user.PasswordHash == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot unlink: no password set. Please set a password first.",
		})
	}

	user.GoogleID = ""
	database.DB.Save(&user)

	return c.JSON(fiber.Map{
		"message": "Google account unlinked successfully",
	})
}

// GetUserGoogleStatus returns if user has Google linked
func GetUserGoogleStatus(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	return c.JSON(fiber.Map{
		"google_linked": user.GoogleID != "",
		"avatar_url":    user.AvatarURL,
	})
}

func getGoogleUserInfo(accessToken string) (*GoogleUserInfo, error) {
	resp, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + accessToken)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var userInfo GoogleUserInfo
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return nil, err
	}

	return &userInfo, nil
}
