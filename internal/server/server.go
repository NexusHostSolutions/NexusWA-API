package server

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/nexus/gowhats/config"
	"github.com/nexus/gowhats/internal/handlers"
	"github.com/nexus/gowhats/internal/middleware"
	"github.com/nexus/gowhats/internal/whatsapp"
)

func NewServer(cfg *config.Config) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName: "NexusWA API - High Performance WhatsApp Engine",
	})

	// Configuração do CORS (Permite que o Frontend acesse a API)
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, apikey",
	}))
	
	app.Use(logger.New())

	// Inicialização de Serviços
	waService := whatsapp.NewService()

	// Handlers
	sessionHandler := handlers.NewSessionHandler(waService)
	msgHandler := handlers.NewMessageHandler(waService)
	groupHandler := handlers.NewGroupHandler(waService)

	// --- AQUI ESTÁ A MUDANÇA ---
	// Removemos o app.Get("/") antigo que mostrava o JSON
	// E colocamos isto para mostrar a pasta public:
	app.Static("/", "./public")

	// Grupo de rotas API v1
	v1 := app.Group("/v1")
	
	// Middleware de Autenticação Global
	v1.Use(middleware.Protected(cfg))

	// Rotas de Instância/Sessão
	instance := v1.Group("/instance")
	instance.Post("/:instance/connect", sessionHandler.Connect)
	instance.Post("/:instance/logout", sessionHandler.Logout)

	// Rotas de Mensagem
	msg := v1.Group("/message")
	msg.Post("/:instance/text", msgHandler.SendText)
	msg.Post("/:instance/interactive", msgHandler.SendInteractive)

	// Rotas de Grupo
	grp := v1.Group("/group")
	grp.Post("/:instance/create", groupHandler.Create)
	grp.Put("/:instance/:group_id/update", groupHandler.UpdateParticipants)

	return app
}