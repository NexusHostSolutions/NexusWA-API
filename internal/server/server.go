package server

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/jmoiron/sqlx"
	"encoding/json" // <-- Faltava

	// imports reais
	"github.com/nexus/gowhats/internal/whatsapp"
	"github.com/nexus/gowhats/internal/handlers"
	"github.com/nexus/gowhats/internal/models"
)

type Server struct {
	app       *fiber.App
	client    *whatsapp.BaileysClient   // ✔ tipo certo
	db        *sqlx.DB
	dbHandler *handlers.DatabaseHandler // ✔ vindo do handlers
}

func NewServer(db *sqlx.DB) *Server {
	app := fiber.New(fiber.Config{
		BodyLimit: 50 * 1024 * 1024,
	})

	app.Use(cors.New())
	app.Use(logger.New())

	client := whatsapp.NewBaileysClient(baileysURL)
	dbHandler := database.NewDatabaseHandler(db, client)


	server := &Server{
		app:       app,
		client:    client,
		db:        db,
		dbHandler: dbHandler,
	}

	server.setupRoutes()

	return server
}

func (s *Server) setupRoutes() {
	// Rota pública (health check)
	s.app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// Todas as rotas /v1 requerem autenticação
	api := s.app.Group("/v1", s.dbHandler.AuthMiddleware)

	// Rota de informações do usuário
	api.Get("/me", s.dbHandler.GetMe)

	// ============================================
	// ROTAS DE API KEYS (Super Admin Only)
	// ============================================
	apiKeys := api.Group("/api-keys", s.dbHandler.SuperAdminOnly)
	apiKeys.Get("/", s.dbHandler.ListApiKeys)
	apiKeys.Post("/", s.dbHandler.CreateApiKey)
	apiKeys.Put("/:id", s.dbHandler.UpdateApiKey)
	apiKeys.Delete("/:id", s.dbHandler.DeleteApiKey)

	// ============================================
	// ROTAS DE INSTÂNCIAS
	// ============================================
	api.Get("/instances", s.dbHandler.ListInstances)
	api.Post("/instances", s.dbHandler.SuperAdminOnly, s.dbHandler.CreateInstance)
	api.Delete("/instances/:name", s.dbHandler.SuperAdminOnly, s.dbHandler.DeleteInstance)

	// ============================================
	// ROTAS DE INSTÂNCIA ESPECÍFICA
	// ============================================
	instance := api.Group("/instance/:instance", s.dbHandler.InstanceAccessMiddleware)
	
	instance.Get("/info", func(c *fiber.Ctx) error {
		instName := c.Params("instance")
		info, err := s.client.GetInstanceInfo(instName)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(info)
	})

	instance.Get("/sync-status", func(c *fiber.Ctx) error {
		instName := c.Params("instance")
		status, err := s.client.GetSyncStatus(instName)
		if err != nil {
			return c.JSON(fiber.Map{"syncing": false, "completed": false})
		}
		return c.JSON(status)
	})

	instance.Post("/sync", func(c *fiber.Ctx) error {
		instName := c.Params("instance")
		err := s.client.TriggerSync(instName)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"status": "started"})
	})

	// ============================================
	// ROTAS DE BANCO DE DADOS
	// ============================================
	db := api.Group("/db")
	db.Get("/contacts/:instance", s.dbHandler.InstanceAccessMiddleware, s.dbHandler.GetContacts)
	db.Get("/groups/:instance", s.dbHandler.InstanceAccessMiddleware, s.dbHandler.GetGroups)
	db.Get("/stats/:instance", s.dbHandler.InstanceAccessMiddleware, s.dbHandler.GetStats)

	// ============================================
	// ROTAS DE CONTATOS E GRUPOS (via Node.js)
	// ============================================
	api.Get("/contacts/:instance", s.dbHandler.InstanceAccessMiddleware, func(c *fiber.Ctx) error {
		instName := c.Params("instance")
		contacts, err := s.client.GetContacts(instName)
		if err != nil {
			return c.JSON([]interface{}{})
		}
		return c.JSON(contacts)
	})

	api.Get("/groups/:instance", s.dbHandler.InstanceAccessMiddleware, func(c *fiber.Ctx) error {
		instName := c.Params("instance")
		groups, err := s.client.GetGroups(instName)
		if err != nil {
			return c.JSON([]interface{}{})
		}
		return c.JSON(groups)
	})

	// ============================================
	// ROTAS DE MENSAGENS
	// ============================================
	message := api.Group("/message")

	message.Post("/text", func(c *fiber.Ctx) error {
		var body map[string]interface{}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "JSON inválido"})
		}

		// Verifica acesso à instância
		instName, _ := body["instance"].(string)
		if !s.checkInstanceAccess(c, instName) {
			return c.Status(403).JSON(fiber.Map{"error": "Sem permissão para esta instância"})
		}

		result, err := s.client.SendText(
	    body["instance"].(string),
    	body["number"].(string),
    	body["text"].(string),
    	nil,
		)

		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(result)
	})

	message.Post("/buttons", func(c *fiber.Ctx) error {
		var body map[string]interface{}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "JSON inválido"})
		}

		instName, _ := body["instance"].(string)
		if !s.checkInstanceAccess(c, instName) {
			return c.Status(403).JSON(fiber.Map{"error": "Sem permissão para esta instância"})
		}

		result, err := s.client.SendButtons(
    		body["instance"].(string),
    		body["number"].(string),
    		body["message"].(string),
    		body["footer"].(string),
    		body["title"].(string),
    		body["buttons"].([]map[string]string),
		)

		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(result)
	})

	message.Post("/list", func(c *fiber.Ctx) error {
		var body map[string]interface{}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "JSON inválido"})
		}

		instName, _ := body["instance"].(string)
		if !s.checkInstanceAccess(c, instName) {
			return c.Status(403).JSON(fiber.Map{"error": "Sem permissão para esta instância"})
		}

		result, err := s.client.SendList(
	    	body["instance"].(string),
    		body["number"].(string),
    		body["title"].(string),
    		body["message"].(string),
    		body["footer"].(string),
    		body["buttonText"].(string),
    		body["sections"].([]map[string]interface{}),
		)

		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(result)
	})

	message.Post("/url-button", func(c *fiber.Ctx) error {
		var body map[string]interface{}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "JSON inválido"})
		}

		instName, _ := body["instance"].(string)
		if !s.checkInstanceAccess(c, instName) {
			return c.Status(403).JSON(fiber.Map{"error": "Sem permissão para esta instância"})
		}

		result, err := s.client.SendUrlButton(
    		body["instance"].(string),
    		body["number"].(string),
    		body["message"].(string),
    		body["footer"].(string),
    		body["title"].(string),
    		body["buttonText"].(string),
    		body["url"].(string),
	)

		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(result)
	})

	message.Post("/copy-button", func(c *fiber.Ctx) error {
		var body map[string]interface{}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "JSON inválido"})
		}

		instName, _ := body["instance"].(string)
		if !s.checkInstanceAccess(c, instName) {
			return c.Status(403).JSON(fiber.Map{"error": "Sem permissão para esta instância"})
		}

		result, err := s.client.SendCopyButton(
    		body["instance"].(string),
    		body["number"].(string),
    		body["message"].(string),
    		body["footer"].(string),
    		body["title"].(string),
    		body["buttonText"].(string),
    		body["copyCode"].(string),
	)

		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(result)
	})

	message.Post("/interactive", func(c *fiber.Ctx) error {
		var body map[string]interface{}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "JSON inválido"})
		}

		instName, _ := body["instance"].(string)
		if !s.checkInstanceAccess(c, instName) {
			return c.Status(403).JSON(fiber.Map{"error": "Sem permissão para esta instância"})
		}

		interactive := &models.InteractivePayload{}
			json.Unmarshal(c.Body(), &interactive)

	result, err := s.client.SendInteractive(
    	body["instance"].(string),
    	body["number"].(string),
    	interactive,
    	nil,
		)

		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(result)
	})

	// ============================================
	// ROTAS DE SESSÃO (proxy para Node.js)
	// ============================================
	session := s.app.Group("/session", s.dbHandler.AuthMiddleware)

	session.Post("/start", func(c *fiber.Ctx) error {
		var body map[string]interface{}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "JSON inválido"})
		}

		instName, _ := body["instance"].(string)
		if !s.checkInstanceAccess(c, instName) {
			return c.Status(403).JSON(fiber.Map{"error": "Sem permissão para esta instância"})
		}

		result, err := s.client.StartSession(body)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(result)
	})

	session.Post("/pair-code", func(c *fiber.Ctx) error {
		var body map[string]interface{}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "JSON inválido"})
		}

		instName, _ := body["instance"].(string)
		if !s.checkInstanceAccess(c, instName) {
			return c.Status(403).JSON(fiber.Map{"error": "Sem permissão para esta instância"})
		}

		result, err := s.client.RequestPairCode(body)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(result)
	})

	session.Post("/logout", func(c *fiber.Ctx) error {
		var body map[string]interface{}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "JSON inválido"})
		}

		instName, _ := body["instance"].(string)
		if !s.checkInstanceAccess(c, instName) {
			return c.Status(403).JSON(fiber.Map{"error": "Sem permissão para esta instância"})
		}

		result, err := s.client.Logout(body)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(result)
	})

	// Arquivos estáticos
	s.app.Static("/", "./public")
}

func (s *Server) checkInstanceAccess(c *fiber.Ctx, instName string) bool {
	if instName == "" {
		return true
	}

	isSuperAdmin, _ := c.Locals("isSuperAdmin").(bool)
	if isSuperAdmin {
		return true
	}

	apiKey, ok := c.Locals("apiKey").(*models.ApiKey)
	if !ok {
		return false
	}

	for _, inst := range apiKey.InstanciasPermitidas {
		if inst == instName {
			return true
		}
	}

	return false
}

func (s *Server) Listen(addr string) error {
	return s.app.Listen(addr)
}