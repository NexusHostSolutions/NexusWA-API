package server

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/nexus/gowhats/config"
	"github.com/nexus/gowhats/internal/handlers"
	"github.com/nexus/gowhats/internal/middleware"
	"github.com/nexus/gowhats/internal/models"
	"github.com/nexus/gowhats/internal/whatsapp"
)

func NewServer(cfg *config.Config) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:   "NexusWA API - Enterprise",
		BodyLimit: 50 * 1024 * 1024,
	})

	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, apikey",
	}))
	app.Use(logger.New())

	waService := whatsapp.NewService()
	sessionHandler := handlers.NewSessionHandler(waService)
	msgHandler := handlers.NewMessageHandler(waService)
	groupHandler := handlers.NewGroupHandler(waService)
	
	// 🆕 Handler de Banco de Dados
	dbHandler := handlers.NewDatabaseHandler(waService)

	app.Static("/", "./public")

	v1 := app.Group("/v1")
	v1.Use(middleware.Protected(cfg))

	// === INSTÂNCIA ===
	v1.Post("/instance/:instance/connect", sessionHandler.Connect)
	v1.Post("/instance/:instance/logout", sessionHandler.Logout)
	v1.Get("/instance/:instance/info", func(c *fiber.Ctx) error {
		info, err := waService.GetInstanceInfo(c.Params("instance"))
		if err != nil {
			return c.JSON(fiber.Map{"status": "disconnected", "error": err.Error()})
		}
		return c.JSON(info)
	})

	// === PAREAMENTO ===
	v1.Post("/instance/:instance/pair", func(c *fiber.Ctx) error {
		type PairReq struct{ Phone string `json:"phone"` }
		var req PairReq
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "JSON inválido"})
		}

		code, err := waService.PairPhone(c.Params("instance"), req.Phone)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		
		return c.JSON(fiber.Map{
			"status": "success", 
			"code": code,
		})
	})

	// === MENSAGENS ===
	v1.Post("/message/:instance/text", msgHandler.SendText)
	v1.Post("/message/:instance/interactive", msgHandler.SendInteractive)

	v1.Get("/messages/:instance/:jid", func(c *fiber.Ctx) error {
		instance := c.Params("instance")
		jid := c.Params("jid")
		
		messages, err := waService.GetMessages(instance, jid)
		if err != nil {
			return c.JSON([]interface{}{})
		}
		return c.JSON(messages)
	})

	// === GRUPOS ===
	v1.Post("/group/:instance/create", groupHandler.Create)
	v1.Put("/group/:instance/:group_id/update", groupHandler.UpdateParticipants)
	
	v1.Get("/group/:instance/list", func(c *fiber.Ctx) error {
		groups, err := waService.GetGroups(c.Params("instance"))
		if err != nil { return c.JSON([]interface{}{}) }
		return c.JSON(groups)
	})

	// === CHAT ===
	v1.Get("/chat/:instance/contacts", func(c *fiber.Ctx) error {
		contacts, err := waService.GetContacts(c.Params("instance"))
		if err != nil { return c.JSON([]interface{}{}) }
		return c.JSON(contacts)
	})

	v1.Get("/chat/:instance/search", func(c *fiber.Ctx) error {
		query := c.Query("q")
		if query == "" { return c.Status(400).JSON(fiber.Map{"error": "Query required"}) }
		results, err := waService.SearchContacts(c.Params("instance"), query)
		if err != nil { return c.JSON([]interface{}{}) }
		return c.JSON(results)
	})

	v1.Post("/chat/:instance/send", func(c *fiber.Ctx) error {
		type ChatSendReq struct { To string `json:"to"`; Text string `json:"text"` }
		var req ChatSendReq
		if err := c.BodyParser(&req); err != nil { return c.Status(400).JSON(fiber.Map{"error": "Invalid JSON"}) }
		msgReq := models.SendMessageRequest{ Number: req.To, Type: "text", Text: req.Text }
		msgID, err := waService.SendMessage(c.Params("instance"), msgReq)
		if err != nil { return c.Status(500).JSON(fiber.Map{"error": err.Error()}) }
		return c.JSON(fiber.Map{"status": "success", "messageId": msgID})
	})

	v1.Post("/settings/:instance/save", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "success", "msg": "Salvo"})
	})

	// ============================================
	// 🆕 ROTAS DE BANCO DE DADOS (NOVAS)
	// ============================================
	
	v1.Get("/db/contacts/:instance", dbHandler.GetContacts)
	v1.Get("/db/groups/:instance", dbHandler.GetGroups)
	v1.Get("/db/messages/:instance/:jid", dbHandler.GetMessages)

	// ============================================

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "version": "3.0-baileys"})
	})

	return app
}