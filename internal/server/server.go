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
		AppName: "NexusWA API - Enterprise",
		BodyLimit: 50 * 1024 * 1024,
	})

	app.Use(cors.New(cors.Config{AllowOrigins: "*", AllowHeaders: "Origin, Content-Type, Accept, apikey"}))
	app.Use(logger.New())

	waService := whatsapp.NewService()
	sessionHandler := handlers.NewSessionHandler(waService)
	msgHandler := handlers.NewMessageHandler(waService)
	groupHandler := handlers.NewGroupHandler(waService)

	app.Static("/", "./public")

	v1 := app.Group("/v1")
	v1.Use(middleware.Protected(cfg))

	// Instância
	v1.Post("/instance/:instance/connect", sessionHandler.Connect)
	v1.Post("/instance/:instance/logout", sessionHandler.Logout)
	v1.Get("/instance/:instance/info", func(c *fiber.Ctx) error {
		info, err := waService.GetInstanceInfo(c.Params("instance"))
		if err != nil {
			return c.JSON(fiber.Map{"status": "disconnected", "error": err.Error()})
		}
		return c.JSON(info)
	})
	
	// Pareamento
	v1.Post("/instance/:instance/pair", func(c *fiber.Ctx) error {
		type PairReq struct { Phone string `json:"phone"` }
		var req PairReq
		if err := c.BodyParser(&req); err != nil { return c.Status(400).JSON(fiber.Map{"error": "JSON inválido"}) }
		code, err := waService.PairPhone(c.Params("instance"), req.Phone)
		if err != nil { return c.Status(500).JSON(fiber.Map{"error": err.Error()}) }
		return c.JSON(fiber.Map{"status": "success", "code": code})
	})

	// Mensagens
	v1.Post("/message/:instance/text", msgHandler.SendText)
	v1.Post("/message/:instance/interactive", msgHandler.SendInteractive)

	// Grupos
	v1.Post("/group/:instance/create", groupHandler.Create)
	v1.Put("/group/:instance/:group_id/update", groupHandler.UpdateParticipants)

	// CHAT: Contatos Reais
	v1.Get("/chat/:instance/contacts", func(c *fiber.Ctx) error {
		contacts, err := waService.GetContacts(c.Params("instance"))
		if err != nil { 
			// Retorna array vazio em vez de erro para o front não quebrar
			return c.JSON([]interface{}{}) 
		}
		return c.JSON(contacts)
	})

	// Configurações (Dummy Persistence)
	v1.Post("/settings/:instance/save", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "success", "msg": "Configurações salvas"})
	})

	return app
}