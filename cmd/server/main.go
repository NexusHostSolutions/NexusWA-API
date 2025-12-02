package main

import (
	"log"

	"github.com/nexus/gowhats/config"
	"github.com/nexus/gowhats/internal/server"
)

func main() {
	// Carregar Configurações
	cfg := config.LoadConfig()

	// Iniciar Servidor
	app := server.NewServer(cfg)

	log.Printf("🔥 NexusWA Iniciando na porta %s...", cfg.ServerPort)
	if err := app.Listen(":" + cfg.ServerPort); err != nil {
		log.Fatal(err)
	}
}