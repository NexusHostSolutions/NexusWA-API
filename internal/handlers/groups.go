package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/nexus/gowhats/internal/models"
	"github.com/nexus/gowhats/internal/whatsapp"
)

type GroupHandler struct {
	Service *whatsapp.Service
}

func NewGroupHandler(s *whatsapp.Service) *GroupHandler {
	return &GroupHandler{Service: s}
}

func (h *GroupHandler) Create(c *fiber.Ctx) error {
	instanceKey := c.Params("instance")
	var req models.CreateGroupRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid JSON"})
	}

	gid, err := h.Service.CreateGroup(instanceKey, req)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"status": "success", "group_id": gid})
}

func (h *GroupHandler) UpdateParticipants(c *fiber.Ctx) error {
	instanceKey := c.Params("instance")
	groupID := c.Params("group_id")
	
	var req models.GroupActionRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid JSON"})
	}
	req.GroupID = groupID

	action := c.Query("action")
	if action == "" {
		return c.Status(400).JSON(fiber.Map{"error": "action required"})
	}

	err := h.Service.ManageGroup(instanceKey, action, req)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"status": "success"})
}