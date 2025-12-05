package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/nexus/gowhats/internal/whatsapp"
)

type DatabaseHandler struct {
	Service *whatsapp.Service
}

func NewDatabaseHandler(s *whatsapp.Service) *DatabaseHandler {
	return &DatabaseHandler{Service: s}
}

func (h *DatabaseHandler) GetContacts(c *fiber.Ctx) error {
	instance := c.Params("instance")
	
	contacts, err := h.Service.Client.GetContactsDB(instance)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error":   err.Error(),
			"message": "Erro ao buscar contatos do banco",
		})
	}
	
	if contacts == nil {
		contacts = []map[string]interface{}{}
	}
	
	return c.JSON(fiber.Map{
		"status": "success",
		"count":  len(contacts),
		"data":   contacts,
	})
}

func (h *DatabaseHandler) GetGroups(c *fiber.Ctx) error {
	instance := c.Params("instance")
	
	groups, err := h.Service.Client.GetGroupsDB(instance)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error":   err.Error(),
			"message": "Erro ao buscar grupos do banco",
		})
	}
	
	if groups == nil {
		groups = []map[string]interface{}{}
	}
	
	return c.JSON(fiber.Map{
		"status": "success",
		"count":  len(groups),
		"data":   groups,
	})
}

func (h *DatabaseHandler) GetMessages(c *fiber.Ctx) error {
	instance := c.Params("instance")
	jid := c.Params("jid")
	
	messages, err := h.Service.Client.GetMessagesDB(instance, jid)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error":   err.Error(),
			"message": "Erro ao buscar mensagens do banco",
		})
	}
	
	if messages == nil {
		messages = []map[string]interface{}{}
	}
	
	return c.JSON(fiber.Map{
		"status": "success",
		"count":  len(messages),
		"data":   messages,
	})
}