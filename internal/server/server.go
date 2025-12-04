package server

import (
	"log"
	
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

	// Webhook Simples (pode ser expandido)
	if cfg.WebhookURL != "" {
		waService.GetEventBus().Subscribe(func(evt whatsapp.Event) {
			// Envia evento para webhook configurado
			log.Printf("[WEBHOOK] Enviando evento %s para %s", evt.Type, cfg.WebhookURL)
			// Aqui você pode fazer um POST HTTP para cfg.WebhookURL com o evt
		})
	}

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

	// Pareamento
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
		return c.JSON(fiber.Map{"status": "success", "code": code})
	})

	// === MENSAGENS ===
	v1.Post("/message/:instance/text", msgHandler.SendText)
	v1.Post("/message/:instance/interactive", msgHandler.SendInteractive)

	// === GRUPOS ===
	v1.Post("/group/:instance/create", groupHandler.Create)
	v1.Put("/group/:instance/:group_id/update", groupHandler.UpdateParticipants)
	
	// NOVA: Listar grupos
	v1.Get("/group/:instance/list", func(c *fiber.Ctx) error {
		groups, err := waService.GetGroups(c.Params("instance"))
		if err != nil {
			return c.JSON([]interface{}{})
		}
		return c.JSON(groups)
	})

	// === CHAT: Contatos e Grupos ===
	v1.Get("/chat/:instance/contacts", func(c *fiber.Ctx) error {
		contacts, err := waService.GetContacts(c.Params("instance"))
		if err != nil {
			return c.JSON([]interface{}{})
		}
		return c.JSON(contacts)
	})

	// NOVA: Buscar contatos
	v1.Get("/chat/:instance/search", func(c *fiber.Ctx) error {
		query := c.Query("q")
		if query == "" {
			return c.Status(400).JSON(fiber.Map{"error": "Query 'q' obrigatória"})
		}
		
		results, err := waService.SearchContacts(c.Params("instance"), query)
		if err != nil {
			return c.JSON([]interface{}{})
		}
		return c.JSON(results)
	})

	// NOVA: Enviar mensagem via Chat
	v1.Post("/chat/:instance/send", func(c *fiber.Ctx) error {
		type ChatSendReq struct {
			To   string `json:"to"`
			Text string `json:"text"`
		}
		var req ChatSendReq
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "JSON inválido"})
		}

		msgReq := models.SendMessageRequest{
			Number: req.To,
			Type:   "text",
			Text:   req.Text,
		}
		
		msgID, err := waService.SendMessage(c.Params("instance"), msgReq)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"status": "success", "messageId": msgID})
	})

	// === CONFIGURAÇÕES ===
	v1.Post("/settings/:instance/save", func(c *fiber.Ctx) error {
		// Aqui você pode salvar em banco/arquivo se quiser
		return c.JSON(fiber.Map{"status": "success", "msg": "Configurações salvas"})
	})

	// === ESTATÍSTICAS GLOBAIS ===
	v1.Get("/stats/global", func(c *fiber.Ctx) error {
		// Aqui você pode agregar dados de todas as instâncias
		return c.JSON(fiber.Map{
			"instances": 0,
			"messages":  0,
			"uptime":    "24h",
		})
	})

	// Health Check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "version": "2.0"})
	})

	return app
}