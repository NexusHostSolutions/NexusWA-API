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
	
	// Pede conexão ao motor
	qrChan, err := h.Service.Client.Connect(instanceKey)
	
	if err != nil {
		if err.Error() == "already_connected" {
			return c.JSON(fiber.Map{"status": "success", "message": "Already connected", "qrcode": ""})
		}
		if err.Error() == "session_restored" {
			return c.JSON(fiber.Map{"status": "success", "message": "Session restored from database", "qrcode": ""})
		}
		return c.Status(500).JSON(fiber.Map{"status": "error", "message": err.Error()})
	}

	// Aguarda o código do WhatsApp
	select {
	case code := <-qrChan:
		// DEBUG: Imprime o código no terminal caso a imagem falhe
		fmt.Printf("\n>>> CÓDIGO RAW (%s): %s\n\n", instanceKey, code)

		// GERA A IMAGEM AQUI MESMO
		// Mudamos para qrcode.Low (Nível Baixo de recuperação) -> Isso faz os pontos ficarem MAIORES e mais fáceis de ler
		// Aumentamos o tamanho para 512px
		png, err := qrcode.Encode(code, qrcode.Low, 512)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Failed to generate QR image"})
		}

		// Transforma em Base64 para o HTML ler direto
		qrBase64 := "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)
		
		return c.JSON(fiber.Map{
			"status":   "success",
			"instance": instanceKey,
			"qrcode":   qrBase64,
		})

	case <-time.After(15 * time.Second):
		return c.Status(408).JSON(fiber.Map{"status": "error", "message": "Timeout waiting for WhatsApp QR Code"})
	}
}

func (h *SessionHandler) Logout(c *fiber.Ctx) error {
	instanceKey := c.Params("instance")
	h.Service.Client.Logout(instanceKey)
	return c.JSON(fiber.Map{"status": "success", "message": "Logged out"})
}