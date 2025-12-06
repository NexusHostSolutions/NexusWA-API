package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jmoiron/sqlx"
	"github.com/nexus/gowhats/internal/whatsapp" // <-- IMPORT CORRETO
)

type DatabaseHandler struct {
	db     *sqlx.DB
	client *whatsapp.BaileysClient // <-- Ajustado
}

func NewDatabaseHandler(db *sqlx.DB, client *whatsapp.BaileysClient) *DatabaseHandler { // <-- Ajustado
	return &DatabaseHandler{
		db:     db,
		client: client,
	}
}

// ============================================
// STRUCTS
// ============================================

type ApiKey struct {
	ID                   int        `db:"id" json:"id"`
	KeyHash              string     `db:"key_hash" json:"-"`
	KeyPrefix            string     `db:"key_prefix" json:"prefix"`
	Nome                 string     `db:"nome" json:"nome"`
	Tipo                 string     `db:"tipo" json:"tipo"`
	Ativa                bool       `db:"ativa" json:"ativa"`
	InstanciasPermitidas []string   `db:"instancias_permitidas" json:"instanciasPermitidas"`
	CriadoEm             time.Time  `db:"criado_em" json:"criadoEm"`
	ExpiraEm             *time.Time `db:"expira_em" json:"expiraEm"`
	UltimoUso            *time.Time `db:"ultimo_uso" json:"ultimoUso"`
	TotalRequisicoes     int        `db:"total_requisicoes" json:"totalRequisicoes"`
}

type ApiKeyValidation struct {
	Valid        bool
	Error        string
	ApiKey       *ApiKey
	IsSuperAdmin bool
}

type Instancia struct {
	ID          int        `db:"id" json:"id"`
	Nome        string     `db:"nome" json:"name"`
	Status      string     `db:"status" json:"status"`
	PushName    *string    `db:"push_name" json:"pushName"`
	Jid         *string    `db:"jid" json:"jid"`
	Avatar      *string    `db:"avatar" json:"avatar"`
	WebhookURL  *string    `db:"webhook_url" json:"webhookUrl"`
	WebhookOn   bool       `db:"webhook_enabled" json:"webhookEnabled"`
	CriadoPor   *int       `db:"criado_por" json:"criadoPor"`
	CriadoEm    time.Time  `db:"criado_em" json:"createdAt"`
	AtualizadoEm time.Time `db:"atualizado_em" json:"updatedAt"`
}

type Contato struct {
	ID       int       `db:"id" json:"id"`
	Instance string    `db:"instance" json:"instance"`
	Jid      string    `db:"jid" json:"jid"`
	Nome     string    `db:"nome" json:"name"`
	CriadoEm time.Time `db:"criado_em" json:"createdAt"`
}

type Grupo struct {
	ID       int       `db:"id" json:"id"`
	Instance string    `db:"instance" json:"instance"`
	Jid      string    `db:"jid" json:"jid"`
	Nome     string    `db:"nome" json:"name"`
	CriadoEm time.Time `db:"criado_em" json:"createdAt"`
}

type DBStats struct {
	Contacts int `json:"contacts"`
	Groups   int `json:"groups"`
	Messages int `json:"messages"`
}

// ============================================
// FUNÇÕES DE HASH
// ============================================

func hashApiKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

func getKeyPrefix(key string) string {
	if len(key) >= 8 {
		return key[:8]
	}
	return key
}

// ============================================
// VALIDAÇÃO DE API KEY
// ============================================

func (h *DatabaseHandler) ValidateApiKey(key string) ApiKeyValidation {
	if key == "" {
		return ApiKeyValidation{Valid: false, Error: "API Key não fornecida"}
	}

	keyHash := hashApiKey(key)

	var apiKey ApiKey
	err := h.db.Get(&apiKey, `
		SELECT id, key_hash, key_prefix, nome, tipo, ativa, instancias_permitidas, 
		       criado_em, expira_em, ultimo_uso, total_requisicoes
		FROM api_keys WHERE key_hash = $1
	`, keyHash)

	if err != nil {
		return ApiKeyValidation{Valid: false, Error: "API Key inválida"}
	}

	if !apiKey.Ativa {
		return ApiKeyValidation{Valid: false, Error: "API Key desativada"}
	}

	if apiKey.ExpiraEm != nil && apiKey.ExpiraEm.Before(time.Now()) {
		return ApiKeyValidation{Valid: false, Error: "API Key expirada"}
	}

	// Atualiza último uso
	h.db.Exec(`UPDATE api_keys SET ultimo_uso = NOW(), total_requisicoes = total_requisicoes + 1 WHERE id = $1`, apiKey.ID)

	return ApiKeyValidation{
		Valid:        true,
		ApiKey:       &apiKey,
		IsSuperAdmin: apiKey.Tipo == "super_admin",
	}
}

// ============================================
// MIDDLEWARE DE AUTENTICAÇÃO
// ============================================

func (h *DatabaseHandler) AuthMiddleware(c *fiber.Ctx) error {
	apiKey := c.Get("apikey")
	if apiKey == "" {
		apiKey = c.Get("x-api-key")
	}
	if apiKey == "" {
		apiKey = c.Query("apikey")
	}

	validation := h.ValidateApiKey(apiKey)
	if !validation.Valid {
		return c.Status(401).JSON(fiber.Map{"error": validation.Error})
	}

	c.Locals("apiKey", validation.ApiKey)
	c.Locals("isSuperAdmin", validation.IsSuperAdmin)

	return c.Next()
}

func (h *DatabaseHandler) SuperAdminOnly(c *fiber.Ctx) error {
	isSuperAdmin, ok := c.Locals("isSuperAdmin").(bool)
	if !ok || !isSuperAdmin {
		return c.Status(403).JSON(fiber.Map{"error": "Acesso restrito a Super Admin"})
	}
	return c.Next()
}

func (h *DatabaseHandler) InstanceAccessMiddleware(c *fiber.Ctx) error {
	instanceName := c.Params("instance")
	if instanceName == "" {
		instanceName = c.Params("name")
	}

	if instanceName == "" {
		return c.Next()
	}

	isSuperAdmin, _ := c.Locals("isSuperAdmin").(bool)
	if isSuperAdmin {
		return c.Next()
	}

	apiKey, ok := c.Locals("apiKey").(*ApiKey)
	if !ok {
		return c.Status(403).JSON(fiber.Map{"error": "Sem permissão"})
	}

	// Verifica se tem acesso à instância
	hasAccess := false
	for _, inst := range apiKey.InstanciasPermitidas {
		if inst == instanceName {
			hasAccess = true
			break
		}
	}

	if !hasAccess {
		return c.Status(403).JSON(fiber.Map{"error": "Sem permissão para esta instância"})
	}

	return c.Next()
}

// ============================================
// ROTAS DE API KEYS
// ============================================

func (h *DatabaseHandler) ListApiKeys(c *fiber.Ctx) error {
	var keys []ApiKey
	err := h.db.Select(&keys, `
		SELECT id, key_prefix, nome, tipo, ativa, instancias_permitidas, 
		       criado_em, expira_em, ultimo_uso, total_requisicoes
		FROM api_keys ORDER BY criado_em DESC
	`)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Erro ao buscar API Keys"})
	}

	// Adiciona campo expirada
	type ApiKeyResponse struct {
		ApiKey
		Expirada bool `json:"expirada"`
	}

	var response []ApiKeyResponse
	for _, k := range keys {
		expirada := false
		if k.ExpiraEm != nil && k.ExpiraEm.Before(time.Now()) {
			expirada = true
		}
		response = append(response, ApiKeyResponse{ApiKey: k, Expirada: expirada})
	}

	return c.JSON(response)
}

func (h *DatabaseHandler) CreateApiKey(c *fiber.Ctx) error {
	var body struct {
		Nome                 string   `json:"nome"`
		Tipo                 string   `json:"tipo"`
		Validade             *string  `json:"validade"`
		InstanciasPermitidas []string `json:"instanciasPermitidas"`
	}

	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "JSON inválido"})
	}

	if body.Nome == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Nome obrigatório"})
	}

	if body.Tipo == "" {
		body.Tipo = "user"
	}

	// Gera a key
	prefix := "nxus_"
	if body.Tipo == "super_admin" {
		prefix = "nxsa_"
	}
	randomBytes := make([]byte, 24)
	for i := range randomBytes {
		randomBytes[i] = byte(time.Now().UnixNano() % 256)
	}
	key := prefix + hex.EncodeToString(randomBytes)
	keyHash := hashApiKey(key)
	keyPrefix := getKeyPrefix(key)

	// Calcula expiração
	var expiraEm *time.Time
	if body.Validade != nil {
		var dias int
		switch *body.Validade {
		case "30":
			dias = 30
		case "90":
			dias = 90
		case "180":
			dias = 180
		case "365":
			dias = 365
		}
		if dias > 0 {
			exp := time.Now().AddDate(0, 0, dias)
			expiraEm = &exp
		}
	}

	var id int
	err := h.db.QueryRow(`
		INSERT INTO api_keys (key_hash, key_prefix, nome, tipo, instancias_permitidas, expira_em)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, keyHash, keyPrefix, body.Nome, body.Tipo, body.InstanciasPermitidas, expiraEm).Scan(&id)

	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Erro ao criar API Key: " + err.Error()})
	}

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "API Key criada! Guarde a key, ela não será mostrada novamente.",
		"apiKey": fiber.Map{
			"id":       id,
			"key":      key,
			"prefix":   keyPrefix,
			"nome":     body.Nome,
			"tipo":     body.Tipo,
			"expiraEm": expiraEm,
		},
	})
}

func (h *DatabaseHandler) UpdateApiKey(c *fiber.Ctx) error {
	id := c.Params("id")

	var body struct {
		Nome                 *string  `json:"nome"`
		Ativa                *bool    `json:"ativa"`
		Tipo                 *string  `json:"tipo"`
		InstanciasPermitidas []string `json:"instanciasPermitidas"`
		RenovarValidade      *string  `json:"renovarValidade"`
	}

	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "JSON inválido"})
	}

	// Monta a query dinamicamente
	updates := []string{}
	args := []interface{}{}
	argIdx := 1

	if body.Nome != nil {
		updates = append(updates, fmt.Sprintf("nome = $%d", argIdx))
		args = append(args, *body.Nome)
		argIdx++
	}

	if body.Ativa != nil {
		updates = append(updates, fmt.Sprintf("ativa = $%d", argIdx))
		args = append(args, *body.Ativa)
		argIdx++
	}

	if body.Tipo != nil {
		updates = append(updates, fmt.Sprintf("tipo = $%d", argIdx))
		args = append(args, *body.Tipo)
		argIdx++
	}

	if body.InstanciasPermitidas != nil {
		updates = append(updates, fmt.Sprintf("instancias_permitidas = $%d", argIdx))
		args = append(args, body.InstanciasPermitidas)
		argIdx++
	}

	if body.RenovarValidade != nil {
		var dias int
		switch *body.RenovarValidade {
		case "30":
			dias = 30
		case "90":
			dias = 90
		case "180":
			dias = 180
		case "365":
			dias = 365
		}
		if dias > 0 {
			exp := time.Now().AddDate(0, 0, dias)
			updates = append(updates, fmt.Sprintf("expira_em = $%d", argIdx))
			args = append(args, exp)
			argIdx++
		}
	}

	if len(updates) == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "Nada para atualizar"})
	}

	args = append(args, id)
	query := fmt.Sprintf("UPDATE api_keys SET %s WHERE id = $%d", 
		joinStrings(updates, ", "), argIdx)

	_, err := h.db.Exec(query, args...)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Erro ao atualizar"})
	}

	return c.JSON(fiber.Map{"status": "success"})
}

func (h *DatabaseHandler) DeleteApiKey(c *fiber.Ctx) error {
	id := c.Params("id")

	// Não permite deletar a própria key
	apiKey, _ := c.Locals("apiKey").(*ApiKey)
	if apiKey != nil && fmt.Sprintf("%d", apiKey.ID) == id {
		return c.Status(400).JSON(fiber.Map{"error": "Não é possível deletar sua própria API Key"})
	}

	_, err := h.db.Exec(`DELETE FROM api_keys WHERE id = $1`, id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Erro ao deletar"})
	}

	return c.JSON(fiber.Map{"status": "success"})
}

// ============================================
// ROTAS DE INSTÂNCIAS
// ============================================

func (h *DatabaseHandler) ListInstances(c *fiber.Ctx) error {
	isSuperAdmin, _ := c.Locals("isSuperAdmin").(bool)
	apiKey, _ := c.Locals("apiKey").(*ApiKey)

	var instances []Instancia
	var err error

	if isSuperAdmin {
		err = h.db.Select(&instances, `SELECT * FROM instancias ORDER BY criado_em DESC`)
	} else if apiKey != nil && len(apiKey.InstanciasPermitidas) > 0 {
		query, args, _ := sqlx.In(`SELECT * FROM instancias WHERE nome IN (?) ORDER BY criado_em DESC`, apiKey.InstanciasPermitidas)
		query = h.db.Rebind(query)
		err = h.db.Select(&instances, query, args...)
	} else {
		instances = []Instancia{}
	}

	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Erro ao buscar instâncias"})
	}

	// Adiciona stats para cada instância
	type InstanceResponse struct {
		Instancia
		Stats DBStats `json:"stats"`
	}

	var response []InstanceResponse
	for _, inst := range instances {
		stats := h.getDBStats(inst.Nome)
		response = append(response, InstanceResponse{Instancia: inst, Stats: stats})
	}

	return c.JSON(response)
}

func (h *DatabaseHandler) CreateInstance(c *fiber.Ctx) error {
	var body struct {
		Name string `json:"name"`
	}

	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "JSON inválido"})
	}

	if body.Name == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Nome obrigatório"})
	}

	apiKey, _ := c.Locals("apiKey").(*ApiKey)
	var criadoPor *int
	if apiKey != nil {
		criadoPor = &apiKey.ID
	}

	_, err := h.db.Exec(`
		INSERT INTO instancias (nome, criado_por)
		VALUES ($1, $2)
		ON CONFLICT (nome) DO UPDATE SET atualizado_em = NOW()
	`, body.Name, criadoPor)

	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Erro ao criar instância"})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"instance": fiber.Map{
			"name":   body.Name,
			"status": "disconnected",
		},
	})
}

func (h *DatabaseHandler) DeleteInstance(c *fiber.Ctx) error {
	name := c.Params("name")

	// Deleta dados relacionados
	h.db.Exec(`DELETE FROM contatos WHERE instance = $1`, name)
	h.db.Exec(`DELETE FROM grupos WHERE instance = $1`, name)
	h.db.Exec(`DELETE FROM mensagens WHERE instance = $1`, name)
	h.db.Exec(`DELETE FROM instancias WHERE nome = $1`, name)

	return c.JSON(fiber.Map{"status": "success"})
}

// ============================================
// ROTAS DE CONTATOS E GRUPOS
// ============================================

func (h *DatabaseHandler) GetContacts(c *fiber.Ctx) error {
	instance := c.Params("instance")

	var contatos []Contato
	err := h.db.Select(&contatos, `
		SELECT id, instance, jid, nome, criado_em 
		FROM contatos WHERE instance = $1 ORDER BY nome ASC
	`, instance)

	if err != nil {
		return c.JSON([]Contato{})
	}

	return c.JSON(contatos)
}

func (h *DatabaseHandler) GetGroups(c *fiber.Ctx) error {
	instance := c.Params("instance")

	var grupos []Grupo
	err := h.db.Select(&grupos, `
		SELECT id, instance, jid, nome, criado_em 
		FROM grupos WHERE instance = $1 ORDER BY nome ASC
	`, instance)

	if err != nil {
		return c.JSON([]Grupo{})
	}

	return c.JSON(grupos)
}

func (h *DatabaseHandler) GetStats(c *fiber.Ctx) error {
	instance := c.Params("instance")
	stats := h.getDBStats(instance)
	return c.JSON(stats)
}

func (h *DatabaseHandler) getDBStats(instance string) DBStats {
	var stats DBStats

	h.db.Get(&stats.Contacts, `SELECT COUNT(*) FROM contatos WHERE instance = $1`, instance)
	h.db.Get(&stats.Groups, `SELECT COUNT(*) FROM grupos WHERE instance = $1`, instance)
	h.db.Get(&stats.Messages, `SELECT COUNT(*) FROM mensagens WHERE instance = $1`, instance)

	return stats
}

// ============================================
// ROTA /v1/me
// ============================================

func (h *DatabaseHandler) GetMe(c *fiber.Ctx) error {
	apiKey, ok := c.Locals("apiKey").(*ApiKey)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"error": "Não autenticado"})
	}

	isSuperAdmin, _ := c.Locals("isSuperAdmin").(bool)

	return c.JSON(fiber.Map{
		"id":                   apiKey.ID,
		"nome":                 apiKey.Nome,
		"tipo":                 apiKey.Tipo,
		"isSuperAdmin":         isSuperAdmin,
		"instanciasPermitidas": apiKey.InstanciasPermitidas,
		"criadoEm":             apiKey.CriadoEm,
		"expiraEm":             apiKey.ExpiraEm,
		"ultimoUso":            apiKey.UltimoUso,
	})
}

// ============================================
// HELPER
// ============================================

func joinStrings(strs []string, sep string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}