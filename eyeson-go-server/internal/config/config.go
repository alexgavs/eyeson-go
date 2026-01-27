package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// MaskPassword скрывает пароль за звёздочками
func MaskPassword(password string) string {
	if len(password) == 0 {
		return ""
	}
	if len(password) <= 2 {
		return strings.Repeat("*", len(password))
	}
	return string(password[0]) + strings.Repeat("*", len(password)-2) + string(password[len(password)-1])
}

type Config struct {
	Port        string
	DBPath      string
	ApiBaseUrl  string
	ApiUsername string
	ApiPassword string
	JwtSecret   string
	ApiDelayMs  int
}

func LoadConfig() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		Port:        getEnv("PORT", "5000"),
		DBPath:      getEnv("DATABASE_PATH", "eyeson.db"),
		ApiBaseUrl:  getEnv("EYESON_API_BASE_URL", "https://eot-portal.pelephone.co.il:8888"),
		ApiUsername: getEnv("EYESON_API_USERNAME", ""),
		ApiPassword: getEnv("EYESON_API_PASSWORD", ""),
		JwtSecret:   getEnv("JWT_SECRET", "change-me-in-prod"),
		ApiDelayMs:  getEnvInt("EYESON_API_DELAY_MS", 1000),
	}

	return cfg, nil
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
