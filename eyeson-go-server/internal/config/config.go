// Copyright (c) 2026 Alexander G.
// Author: Alexander G. (Samsonix)
// License: MIT
// Project: EyesOn SIM Management System

package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv           string
	Port             string
	DBPath           string
	ApiBaseUrl       string
	SimulatorBaseUrl string
	ApiUsername      string
	ApiPassword      string
	JwtSecret        string
	ApiDelayMs       int

	CorsAllowOrigins string
	EnableSwagger    bool

	ApiInsecureTLS bool

	SeedDefaultAdmin     bool
	DefaultAdminPassword string
}

func LoadConfig() (*Config, error) {
	_ = godotenv.Load()

	appEnv := strings.ToLower(strings.TrimSpace(getEnv("APP_ENV", "dev")))
	if appEnv == "" {
		appEnv = "dev"
	}
	if appEnv != "dev" && appEnv != "prod" {
		return nil, fmt.Errorf("invalid APP_ENV: %q (expected dev|prod)", appEnv)
	}

	cfg := &Config{
		AppEnv: appEnv,
		Port:   getEnv("PORT", "5000"),
		DBPath: getEnv("DATABASE_PATH", "eyeson.db"),
		// By default we target the real provider (Pelephone). Simulator can be selected via the Admin UI.
		ApiBaseUrl:       getEnv("EYESON_API_BASE_URL", "https://eot-portal.pelephone.co.il:8888"),
		SimulatorBaseUrl: getEnv("EYESON_SIMULATOR_BASE_URL", "http://127.0.0.1:8888"),
		ApiUsername:      getEnv("EYESON_API_USERNAME", "admin"),
		ApiPassword:      getEnv("EYESON_API_PASSWORD", "admin"),
		JwtSecret:        getEnv("JWT_SECRET", "change-me-in-prod"),
		ApiDelayMs:       getEnvInt("EYESON_API_DELAY_MS", 10),

		CorsAllowOrigins: getEnv("EYESON_CORS_ORIGINS", "http://localhost:5173,http://127.0.0.1:5173,http://localhost:5000,http://127.0.0.1:5000"),
		EnableSwagger:    getEnvBool("EYESON_ENABLE_SWAGGER", appEnv == "dev"),

		ApiInsecureTLS: getEnvBool("EYESON_API_INSECURE_TLS", appEnv == "dev"),

		SeedDefaultAdmin:     getEnvBool("EYESON_SEED_DEFAULT_ADMIN", appEnv == "dev"),
		DefaultAdminPassword: getEnv("EYESON_DEFAULT_ADMIN_PASSWORD", "admin"),
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	// In dev we allow convenience defaults (but other layers should still be safe-by-default).
	if c.AppEnv == "dev" {
		return nil
	}

	// In prod, refuse to start with known insecure defaults.
	if strings.TrimSpace(c.JwtSecret) == "" || c.JwtSecret == "change-me-in-prod" {
		return fmt.Errorf("JWT_SECRET is not set or uses default; refusing to start in prod")
	}
	if c.ApiInsecureTLS {
		return fmt.Errorf("EYESON_API_INSECURE_TLS must be false in prod")
	}
	if c.SeedDefaultAdmin {
		return fmt.Errorf("EYESON_SEED_DEFAULT_ADMIN must be false in prod")
	}
	return nil
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	strValue := getEnv(key, "")
	if strValue == "" {
		return fallback
	}
	val, err := strconv.Atoi(strValue)
	if err != nil {
		return fallback
	}
	return val
}

func getEnvBool(key string, fallback bool) bool {
	strValue := strings.ToLower(strings.TrimSpace(getEnv(key, "")))
	if strValue == "" {
		return fallback
	}
	switch strValue {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return fallback
	}
}
