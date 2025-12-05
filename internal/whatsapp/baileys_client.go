package whatsapp

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/nexus/gowhats/internal/models"

	_ "github.com/lib/pq"
)

type BaileysClient struct {
	BaseURL    string
	HTTPClient *http.Client
	DB         *sql.DB
}

func NewBaileysClient() *BaileysClient {
	client := &BaileysClient{
		BaseURL: "http://localhost:3001",
		HTTPClient: &http.Client{
			Timeout: 50 * time.Second,
		},
	}

	client.initDB()

	return client
}

// ============================================
// 🆕 FUNÇÕES DE BANCO DE DADOS
// ============================================

func (c *BaileysClient) initDB() {
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "")
	user := getEnv("DB_USER", "")
	password := getEnv("DB_PASS", "")
	dbname := getEnv("DB_NAME", "")

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		fmt.Printf("❌ Erro ao conectar PostgreSQL: %v\n", err)
		return
	}

	if err := db.Ping(); err != nil {
		fmt.Printf("❌ Erro ao pingar PostgreSQL: %v\n", err)
		return
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	c.DB = db
	fmt.Println("✅ PostgreSQL conectado (Go)")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func (c *BaileysClient) GetContactsDB(instance string) ([]map[string]interface{}, error) {
	if c.DB == nil {
		return nil, errors.New("banco de dados não conectado")
	}

	rows, err := c.DB.Query(`
		SELECT jid, nome, criado_em 
		FROM contatos 
		WHERE instance = $1 
		ORDER BY nome ASC
	`, instance)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []map[string]interface{}
	for rows.Next() {
		var jid, nome string
		var criadoEm time.Time
		if err := rows.Scan(&jid, &nome, &criadoEm); err != nil {
			continue
		}
		result = append(result, map[string]interface{}{
			"jid":       jid,
			"nome":      nome,
			"criado_em": criadoEm,
		})
	}

	return result, nil
}

func (c *BaileysClient) GetGroupsDB(instance string) ([]map[string]interface{}, error) {
	if c.DB == nil {
		return nil, errors.New("banco de dados não conectado")
	}

	rows, err := c.DB.Query(`
		SELECT jid, nome, criado_em 
		FROM grupos 
		WHERE instance = $1 
		ORDER BY nome ASC
	`, instance)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []map[string]interface{}
	for rows.Next() {
		var jid, nome string
		var criadoEm time.Time
		if err := rows.Scan(&jid, &nome, &criadoEm); err != nil {
			continue
		}
		result = append(result, map[string]interface{}{
			"jid":       jid,
			"nome":      nome,
			"criado_em": criadoEm,
		})
	}

	return result, nil
}

func (c *BaileysClient) GetMessagesDB(instance, jid string) ([]map[string]interface{}, error) {
	if c.DB == nil {
		return nil, errors.New("banco de dados não conectado")
	}

	rows, err := c.DB.Query(`
		SELECT id, tipo, conteudo, criado_em 
		FROM mensagens 
		WHERE instance = $1 AND jid = $2 
		ORDER BY criado_em DESC 
		LIMIT 100
	`, instance, jid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []map[string]interface{}
	for rows.Next() {
		var id int
		var tipo, conteudo string
		var criadoEm time.Time
		if err := rows.Scan(&id, &tipo, &conteudo, &criadoEm); err != nil {
			continue
		}
		result = append(result, map[string]interface{}{
			"id":        id,
			"tipo":      tipo,
			"conteudo":  conteudo,
			"criado_em": criadoEm,
		})
	}

	return result, nil
}

// ============================================
// FUNÇÕES EXISTENTES (NÃO MODIFICADAS)
// ============================================

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
	
	if status, ok := result["status"].(string); ok && status == "success" {
		if key, ok := result["key"].(map[string]interface{}); ok {
			if id, ok := key["id"].(string); ok {
				return id, nil
			}
		}
		return "sent", nil
	}
	
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

func (c *BaileysClient) SendMedia(instance, number string, media *models.MediaPayload, opts *models.MessageOptions) (string, error) {
	return "", nil
}

func (c *BaileysClient) CreateGroup(instance, subject string, participants []string) (string, error) {
	return "", nil
}

func (c *BaileysClient) GroupAction(instance, groupID string, participants []string, action string) error {
	return nil
}