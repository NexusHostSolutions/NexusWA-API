package main

import (
	"log"

	"github.com/nexus/gowhats/config"
	"github.com/nexus/gowhats/internal/server"
)

func main() {
	cfg := config.LoadConfig()

	app := server.NewServer(cfg)

	log.Printf("🚀 NexusWA API Enterprise rodando na porta %s", cfg.ServerPort)
	log.Printf("🔑 API Key: %s", cfg.GlobalApiKey)
	log.Printf("📡 Dashboard: http://localhost:%s", cfg.ServerPort)

	if err := app.Listen(":" + cfg.ServerPort); err != nil {
		log.Fatal(err)
	}
}