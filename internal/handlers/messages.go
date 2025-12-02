package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/nexus/gowhats/internal/models"
	"github.com/nexus/gowhats/internal/whatsapp"
)

type MessageHandler struct {
	Service *whatsapp.Service
}

func NewMessageHandler(s *whatsapp.Service) *MessageHandler {
	return &MessageHandler{Service: s}
}

func (h *MessageHandler) SendText(c *fiber.Ctx) error {
	instanceKey := c.Params("instance")
	var req models.SendMessageRequest
	
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid JSON"})
	}
	req.Type = "text"

	msgID, err := h.Service.SendMessage(instanceKey, req)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"status": "success", "messageId": msgID})
}

// SendInteractive lida com Botões e Listas (Formato 2025)
func (h *MessageHandler) SendInteractive(c *fiber.Ctx) error {
	instanceKey := c.Params("instance")
	var req models.SendMessageRequest
	
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid JSON"})
	}
	req.Type = "interactive"

	// Validação básica
	if req.Interactive == nil {
		return c.Status(400).JSON(fiber.Map{"error": "Interactive payload required"})
	}

	msgID, err := h.Service.SendMessage(instanceKey, req)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"status": "success", "messageId": msgID})
}