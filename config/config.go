package config

import (
	"os"
	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort   string
	GlobalApiKey string
	WebhookURL   string
}

func LoadConfig() *Config {
	_ = godotenv.Load()

	return &Config{
		ServerPort:   getEnv("SERVER_PORT", "8082"),
		GlobalApiKey: getEnv("GLOBAL_API_KEY", "8msyqcp4o7065sz1nxdg8y69kp7gduijvb0zptz867"),
		WebhookURL:   getEnv("WEBHOOK_URL", ""),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}