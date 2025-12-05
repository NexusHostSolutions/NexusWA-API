package whatsapp

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/nexus/gowhats/internal/models"
)

type BaileysClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewBaileysClient() *BaileysClient {
	return &BaileysClient{
		BaseURL: "http://localhost:3001",
		HTTPClient: &http.Client{
			Timeout: 50 * time.Second,
		},
	}
}

// --- CONEXÃO ---

func (c *BaileysClient) Connect(instanceKey string) (<-chan string, error) {
	qrChan := make(chan string, 1)

	go func() {
		defer close(qrChan)
		
		payload := map[string]string{"instance": instanceKey}
		data, _ := json.Marshal(payload)
		
		resp, err := c.HTTPClient.Post(c.BaseURL+"/session/start", "application/json", bytes.NewBuffer(data))
		if err != nil {
			fmt.Printf("Erro ao conectar ao Baileys: %v\n", err)
			return
		}
		defer resp.Body.Close()

		var result struct {
			Status string `json:"status"`
			QRCode string `json:"qrcode"`
			Error  string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			fmt.Printf("Decode error: %v\n", err)
			return
		}

		if result.Error != "" {
			fmt.Println("Erro Baileys:", result.Error)
			return
		}

		if result.Status == "CONNECTED" {
			return 
		}

		if result.Status == "QRCODE" && result.QRCode != "" {
			qrChan <- result.QRCode
		}
	}()

	return qrChan, nil
}

func (c *BaileysClient) PairPhone(instance, phone string) (string, error) {
	payload := map[string]string{
		"instance":    instance,
		"phoneNumber": phone,
	}
	data, _ := json.Marshal(payload)

	resp, err := c.HTTPClient.Post(c.BaseURL+"/session/pair-code", "application/json", bytes.NewBuffer(data))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Status string `json:"status"`
		Code   string `json:"code"`
		Error  string `json:"error"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Error != "" {
		return "", errors.New(result.Error)
	}

	return result.Code, nil
}

func (c *BaileysClient) Logout(instanceKey string) {
	payload := map[string]string{"instance": instanceKey}
	data, _ := json.Marshal(payload)
	c.HTTPClient.Post(c.BaseURL+"/session/logout", "application/json", bytes.NewBuffer(data))
}

// --- INFO ---

func (c *BaileysClient) GetConnectionInfo(instance string) (map[string]interface{}, error) {
	resp, err := c.HTTPClient.Get(fmt.Sprintf("%s/v1/instance/%s/info", c.BaseURL, instance))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, errors.New("disconnected")
	}

	var info map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&info)
	return info, nil
}

func (c *BaileysClient) IsConnected(instanceKey string) bool {
	info, err := c.GetConnectionInfo(instanceKey)
	if err != nil {
		return false
	}
	return info["status"] == "connected"
}

// --- MENSAGENS ---

func (c *BaileysClient) SendText(instance, number, text string, opts *models.MessageOptions) (string, error) {
	return c.postRequest("/v1/message/text", map[string]interface{}{
		"instance": instance, "number": number, "text": text,
	})
}

func (c *BaileysClient) SendInteractive(instance, number string, interactive *models.InteractivePayload, opts *models.MessageOptions) (string, error) {
	return c.postRequest("/v1/message/interactive", map[string]interface{}{
		"instance": instance, "number": number, "interactive": interactive,
	})
}

// 🔥 Enviar botões nativos
func (c *BaileysClient) SendButtons(instance, number, message, footer, title string, buttons []map[string]string) (string, error) {
	return c.postRequest("/v1/message/buttons", map[string]interface{}{
		"instance": instance,
		"number":   number,
		"message":  message,
		"footer":   footer,
		"title":    title,
		"buttons":  buttons,
	})
}

// 🔥 Enviar lista de seleção
func (c *BaileysClient) SendList(instance, number, title, message, footer, buttonText string, sections []map[string]interface{}) (string, error) {
	return c.postRequest("/v1/message/list", map[string]interface{}{
		"instance":   instance,
		"number":     number,
		"title":      title,
		"message":    message,
		"footer":     footer,
		"buttonText": buttonText,
		"sections":   sections,
	})
}

// 🔥 Enviar botão com URL
func (c *BaileysClient) SendUrlButton(instance, number, message, footer, title, buttonText, url string) (string, error) {
	return c.postRequest("/v1/message/url-button", map[string]interface{}{
		"instance":   instance,
		"number":     number,
		"message":    message,
		"footer":     footer,
		"title":      title,
		"buttonText": buttonText,
		"url":        url,
	})
}

// 🔥 Enviar botão de copiar
func (c *BaileysClient) SendCopyButton(instance, number, message, footer, title, buttonText, copyCode string) (string, error) {
	return c.postRequest("/v1/message/copy-button", map[string]interface{}{
		"instance":   instance,
		"number":     number,
		"message":    message,
		"footer":     footer,
		"title":      title,
		"buttonText": buttonText,
		"copyCode":   copyCode,
	})
}

// Buscar mensagens de um chat
func (c *BaileysClient) GetMessages(instance, jid string) ([]map[string]interface{}, error) {
	return c.getRequestList(fmt.Sprintf("/v1/messages/%s/%s", instance, jid))
}

func (c *BaileysClient) GetContacts(instance string) ([]map[string]interface{}, error) {
	return c.getRequestList(fmt.Sprintf("/v1/contacts/%s", instance))
}

func (c *BaileysClient) GetGroups(instance string) ([]map[string]interface{}, error) {
	return c.getRequestList(fmt.Sprintf("/v1/groups/%s", instance))
}

func (c *BaileysClient) SearchContacts(instance, query string) ([]map[string]interface{}, error) {
	contacts, err := c.GetContacts(instance)
	if err != nil {
		return nil, err
	}
	var filtered []map[string]interface{}
	query = strings.ToLower(query)
	for _, contact := range contacts {
		name, _ := contact["name"].(string)
		jid, _ := contact["jid"].(string)
		if query == "" || strings.Contains(strings.ToLower(name), query) || strings.Contains(strings.ToLower(jid), query) {
			filtered = append(filtered, contact)
		}
	}
	return filtered, nil
}

// --- Helpers ---

func (c *BaileysClient) postRequest(endpoint string, payload interface{}) (string, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	resp, err := c.HTTPClient.Post(c.BaseURL+endpoint, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("http error: %d", resp.StatusCode)
	}
	
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	
	// Verifica se tem status de sucesso
	if status, ok := result["status"].(string); ok && status == "success" {
		if key, ok := result["key"].(map[string]interface{}); ok {
			if id, ok := key["id"].(string); ok {
				return id, nil
			}
		}
		return "sent", nil
	}
	
	// Verifica se tem key.id
	if key, ok := result["key"].(map[string]interface{}); ok {
		if id, ok := key["id"].(string); ok {
			return id, nil
		}
	}
	
	return "sent", nil
}

func (c *BaileysClient) getRequestList(endpoint string) ([]map[string]interface{}, error) {
	resp, err := c.HTTPClient.Get(c.BaseURL + endpoint)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	return result, nil
}

// --- Stubs (para compatibilidade) ---

func (c *BaileysClient) SendMedia(instance, number string, media *models.MediaPayload, opts *models.MessageOptions) (string, error) {
	return "", nil
}

func (c *BaileysClient) CreateGroup(instance, subject string, participants []string) (string, error) {
	return "", nil
}

func (c *BaileysClient) GroupAction(instance, groupID string, participants []string, action string) error {
	return nil
}