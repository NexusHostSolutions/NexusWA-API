package config

import (
	
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort string
	GlobalApiKey string
	WebhookURL string
}

func LoadConfig() *Config {
	// Tenta carregar do .env, se não existir segue com env vars do sistema
	_ = godotenv.Load()

	return &Config{
		ServerPort:   getEnv("SERVER_PORT", "8080"),
		GlobalApiKey: getEnv("GLOBAL_API_KEY", "8msyqcp4o7065sz1nxdg8y69kp7gduijvb0zptz867"), // Mudar em produção
		WebhookURL:   getEnv("WEBHOOK_URL", "http://localhost:3000/webhook"),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}