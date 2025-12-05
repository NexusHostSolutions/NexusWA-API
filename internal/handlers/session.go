package handlers

import (
	"encoding/base64"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/nexus/gowhats/internal/whatsapp"
	"github.com/skip2/go-qrcode"
)

type SessionHandler struct {
	Service *whatsapp.Service
}

func NewSessionHandler(s *whatsapp.Service) *SessionHandler {
	return &SessionHandler{Service: s}
}

func (h *SessionHandler) Connect(c *fiber.Ctx) error {
	instanceKey := c.Params("instance")
	
	// Solicita conexão ao Node
	qrChan, err := h.Service.Client.Connect(instanceKey)
	
	if err != nil {
		if err.Error() == "already_connected" {
			return c.JSON(fiber.Map{"status": "success", "message": "Already connected", "qrcode": ""})
		}
		return c.Status(500).JSON(fiber.Map{"status": "error", "message": err.Error()})
	}

	select {
	case code, ok := <-qrChan:
		if !ok || code == "" {
			return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Failed to get QR Code from Node"})
		}
		
		fmt.Printf("\n>>> QR CODE (%s) GERADO <<<\n", instanceKey)

		png, err := qrcode.Encode(code, qrcode.Low, 512)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Failed to generate QR Image"})
		}

		qrBase64 := "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)
		
		return c.JSON(fiber.Map{
			"status":   "success",
			"instance": instanceKey,
			"qrcode":   qrBase64,
		})

	// AUMENTADO PARA 45 SEGUNDOS (para aguentar os retries do Node)
	case <-time.After(45 * time.Second):
		return c.Status(408).JSON(fiber.Map{"status": "error", "message": "Timeout waiting for QR"})
	}
}

func (h *SessionHandler) Logout(c *fiber.Ctx) error {
	instanceKey := c.Params("instance")
	h.Service.Client.Logout(instanceKey)
	return c.JSON(fiber.Map{"status": "success", "message": "Logged out"})
}