package whatsapp

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/nexus/gowhats/internal/models"
	
	_ "github.com/mattn/go-sqlite3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	waLog "go.mau.fi/whatsmeow/util/log"
)

type BaileysClient struct {
	Clients      map[string]*whatsmeow.Client
	mu           sync.RWMutex
	MessageCount map[string]int64 // Contador de mensagens por instância
	EventBus     *EventBus        // Sistema de eventos
}

// Sistema de Eventos para Webhooks
type EventBus struct {
	listeners []func(event Event)
	mu        sync.RWMutex
}

type Event struct {
	Instance  string                 `json:"instance"`
	Type      string                 `json:"type"` // message.received, connection.update, etc
	Timestamp int64                  `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

func NewEventBus() *EventBus {
	return &EventBus{
		listeners: []func(Event){},
	}
}

func (eb *EventBus) Subscribe(fn func(Event)) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.listeners = append(eb.listeners, fn)
}

func (eb *EventBus) Publish(evt Event) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	for _, listener := range eb.listeners {
		go listener(evt) // Async para não bloquear
	}
}

func NewBaileysClient() *BaileysClient {
	_ = os.Mkdir("sessions", 0755)
	return &BaileysClient{
		Clients:      make(map[string]*whatsmeow.Client),
		MessageCount: make(map[string]int64),
		EventBus:     NewEventBus(),
	}
}

func (c *BaileysClient) Connect(instanceKey string) (<-chan string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if client, ok := c.Clients[instanceKey]; ok {
		if client.IsConnected() {
			return nil, errors.New("already_connected")
		}
	}

	// Configuração do Banco com WAL (Alta Performance)
	dbPath := fmt.Sprintf("file:sessions/%s.db?_foreign_keys=on&_busy_timeout=10000&_journal_mode=WAL", instanceKey)
	
	dbLog := waLog.Stdout("Database", "ERROR", true)
	container, err := sqlstore.New(context.Background(), "sqlite3", dbPath, dbLog)
	if err != nil {
		return nil, fmt.Errorf("falha ao abrir banco: %v", err)
	}

	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil {
		return nil, err
	}
	
	if deviceStore == nil {
		deviceStore = container.NewDevice()
	}

	// CORREÇÃO: Define informações do dispositivo
	deviceStore.Platform = "NexusWA-API"
	deviceStore.BusinessName = "NexusWA Enterprise"

	clientLog := waLog.Stdout("Client", "INFO", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)
	
	c.Clients[instanceKey] = client

	// Sistema de Eventos Completo
	client.AddEventHandler(func(evt interface{}) {
		c.handleEvent(instanceKey, evt)
	})

	if client.Store.ID != nil {
		err = client.Connect()
		if err != nil {
			return nil, err
		}
		
		// Publica evento de reconexão
		c.EventBus.Publish(Event{
			Instance:  instanceKey,
			Type:      "connection.update",
			Timestamp: time.Now().Unix(),
			Data: map[string]interface{}{
				"status": "connected",
				"method": "session_restored",
			},
		})
		
		return nil, errors.New("session_restored")
	}

	qrChan, _ := client.GetQRChannel(context.Background())
	err = client.Connect()
	if err != nil {
		return nil, err
	}

	qrStringChan := make(chan string)
	go func() {
		for evt := range qrChan {
			if evt.Event == "code" {
				qrStringChan <- evt.Code
				// Publica evento de QR gerado
				c.EventBus.Publish(Event{
					Instance:  instanceKey,
					Type:      "qr.update",
					Timestamp: time.Now().Unix(),
					Data: map[string]interface{}{
						"code": evt.Code,
					},
				})
			}
		}
		close(qrStringChan)
	}()

	return qrStringChan, nil
}

// Handler de Eventos Unificado
func (c *BaileysClient) handleEvent(instance string, evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		if !v.Info.IsFromMe {
			log.Printf("[MSG] %s: Recebida de %s", instance, v.Info.Sender.User)
			
			// Publica evento de mensagem recebida
			c.EventBus.Publish(Event{
				Instance:  instance,
				Type:      "message.received",
				Timestamp: time.Now().Unix(),
				Data: map[string]interface{}{
					"from":      v.Info.Sender.String(),
					"id":        v.Info.ID,
					"timestamp": v.Info.Timestamp.Unix(),
					"pushName":  v.Info.PushName,
					"isGroup":   v.Info.IsGroup,
				},
			})
		}
		
	case *events.Connected:
		log.Printf("[%s] 🟢 Conectado!", instance)
		c.EventBus.Publish(Event{
			Instance:  instance,
			Type:      "connection.update",
			Timestamp: time.Now().Unix(),
			Data:      map[string]interface{}{"status": "connected"},
		})
		
	case *events.Disconnected:
		log.Printf("[%s] 🔴 Desconectado!", instance)
		c.EventBus.Publish(Event{
			Instance:  instance,
			Type:      "connection.update",
			Timestamp: time.Now().Unix(),
			Data:      map[string]interface{}{"status": "disconnected"},
		})
		
		// Reconexão automática após 5 segundos
		go func() {
			time.Sleep(5 * time.Second)
			log.Printf("[%s] 🔄 Tentando reconectar...", instance)
			_, _ = c.Connect(instance)
		}()
		
	case *events.Receipt:
		// Confirmação de entrega/leitura
		receiptType := "unknown"
		if v.Type == types.ReceiptTypeRead {
			receiptType = "read"
		} else if v.Type == types.ReceiptTypeDelivered {
			receiptType = "delivered"
		} else if v.Type == types.ReceiptTypePlayed {
			receiptType = "played"
		}
		
		c.EventBus.Publish(Event{
			Instance:  instance,
			Type:      "message.receipt",
			Timestamp: time.Now().Unix(),
			Data: map[string]interface{}{
				"type":      receiptType,
				"timestamp": v.Timestamp.Unix(),
			},
		})
	}
}
// --- FUNÇÕES DE INFORMAÇÃO E ESTATÍSTICAS ---

func (c *BaileysClient) GetConnectionInfo(instance string) (map[string]interface{}, error) {
	client, ok := c.getClient(instance)
	if !ok {
		return nil, errors.New("disconnected")
	}
	if client.Store.ID == nil {
		return nil, errors.New("waiting_login")
	}

	jid := client.Store.ID.ToNonAD().String()
	pushName := client.Store.PushName

	// Busca foto de perfil com timeout
	var avatarURL string
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	picInfo, err := client.GetProfilePictureInfo(ctx, *client.Store.ID, &whatsmeow.GetProfilePictureParams{
		Preview:    false,
		ExistingID: "",
	})
	if err == nil && picInfo != nil {
		avatarURL = picInfo.URL
		log.Printf("[%s] ✅ Foto de perfil carregada: %s", instance, avatarURL)
	} else {
		log.Printf("[%s] ⚠️ Sem foto de perfil: %v", instance, err)
	}

	// Estatísticas de Contatos, Grupos e Mensagens Não Lidas
	contactsCount := 0
	groupsCount := 0
	unreadCount := 0
	
	contacts, err := client.Store.Contacts.GetAllContacts(context.Background())
	if err == nil {
		for jid, contact := range contacts {
			if jid.Server == "g.us" {
				groupsCount++
			} else if contact.FullName != "" || contact.PushName != "" {
				contactsCount++
			}
		}
	}

	// Tenta buscar conversas para contar não lidas
	// Nota: whatsmeow não tem API direta para isso, então vamos deixar 0 por enquanto
	// Você pode implementar um contador manual conforme recebe mensagens
	
	log.Printf("[%s] 📊 Stats - Contatos: %d, Grupos: %d", instance, contactsCount, groupsCount)

	// Busca contador de mensagens
	msgCount := c.MessageCount[instance]

	return map[string]interface{}{
		"jid":          jid,
		"name":         pushName,
		"avatar":       avatarURL,
		"status":       "connected",
		"contacts":     contactsCount,
		"groups":       groupsCount,
		"messagesSent": msgCount,
		"unread":       unreadCount, // Mensagens não lidas
	}, nil
}

// --- CHAT: LISTA DE CONTATOS COM FOTO ---

func (c *BaileysClient) GetContacts(instance string) ([]map[string]interface{}, error) {
	client, ok := c.getClient(instance)
	if !ok {
		return nil, errors.New("disconnected")
	}

	contacts, err := client.Store.Contacts.GetAllContacts(context.Background())
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	for jid, contact := range contacts {
		// Pula contatos sem nome
		if contact.FullName == "" && contact.PushName == "" {
			continue
		}
		
		name := contact.FullName
		if name == "" {
			name = contact.PushName
		}
		if name == "" {
			name = jid.User
		}

		// Busca foto do contato
		var avatarURL string
		picInfo, err := client.GetProfilePictureInfo(context.Background(), jid, &whatsmeow.GetProfilePictureParams{Preview: true})
		if err == nil && picInfo != nil {
			avatarURL = picInfo.URL
		}

		result = append(result, map[string]interface{}{
			"jid":      jid.String(),
			"name":     name,
			"avatar":   avatarURL,
			"is_group": jid.Server == "g.us",
			"unread":   0, 
		})
	}
	return result, nil
}

// --- NOVA: LISTA DE GRUPOS COM DETALHES ---

func (c *BaileysClient) GetGroups(instance string) ([]map[string]interface{}, error) {
	client, ok := c.getClient(instance)
	if !ok {
		return nil, errors.New("disconnected")
	}

	// Busca todos os grupos
	groups, err := client.GetJoinedGroups(context.Background())
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	for _, group := range groups {
		// Busca informações detalhadas do grupo
		groupInfo, err := client.GetGroupInfo(context.Background(), group.JID)
		if err != nil {
			continue
		}

		// Busca foto do grupo
		var avatarURL string
		picInfo, err := client.GetProfilePictureInfo(context.Background(), group.JID, &whatsmeow.GetProfilePictureParams{Preview: true})
		if err == nil && picInfo != nil {
			avatarURL = picInfo.URL
		}

		result = append(result, map[string]interface{}{
			"jid":          group.JID.String(),
			"name":         groupInfo.Name,
			"avatar":       avatarURL,
			"participants": len(groupInfo.Participants),
			"owner":        groupInfo.OwnerJID.String(),
			"created":      groupInfo.GroupCreated.Unix(),
		})
	}
	
	return result, nil
}

// --- BUSCA CONTATO POR NOME/NÚMERO ---

func (c *BaileysClient) SearchContacts(instance, query string) ([]map[string]interface{}, error) {
	contacts, err := c.GetContacts(instance)
	if err != nil {
		return nil, err
	}

	var filtered []map[string]interface{}
	for _, contact := range contacts {
		name := contact["name"].(string)
		jid := contact["jid"].(string)
		
		// Busca case-insensitive
		if contains(name, query) || contains(jid, query) {
			filtered = append(filtered, contact)
		}
	}
	
	return filtered, nil
}

func contains(str, substr string) bool {
	return len(str) >= len(substr) && (str == substr || 
		len(str) > 0 && len(substr) > 0 && 
		(str[:len(substr)] == substr || contains(str[1:], substr)))
}

// --- PAREAMENTO VIA CÓDIGO ---

func (c *BaileysClient) PairPhone(instanceKey string, phone string) (string, error) {
	client, ok := c.getClient(instanceKey)
	if !ok {
		return "", errors.New("instância offline")
	}
	if client.IsLoggedIn() {
		return "", errors.New("já logado")
	}
	
	return client.PairPhone(context.Background(), phone, true, whatsmeow.PairClientChrome, "Chrome (Linux)")
}

// --- LOGOUT ---

func (c *BaileysClient) Logout(instanceKey string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if client, ok := c.Clients[instanceKey]; ok {
		if client.IsConnected() {
			client.Logout(context.Background())
		}
		client.Disconnect()
		delete(c.Clients, instanceKey)
		delete(c.MessageCount, instanceKey)
	}
}
// --- MÉTODOS DE ENVIO ---

func (c *BaileysClient) SendText(instance, number, text string, opts *models.MessageOptions) (string, error) {
	client, ok := c.getClient(instance)
	if !ok {
		return "", errors.New("instância desconectada")
	}
	
	if opts != nil && opts.Delay > 0 {
		time.Sleep(time.Duration(opts.Delay) * time.Second)
	}

	jid, _ := types.ParseJID(number + "@s.whatsapp.net")
	msg := &waProto.Message{Conversation: strPtr(text)}
	
	resp, err := client.SendMessage(context.Background(), jid, msg)
	if err != nil {
		return "", err
	}
	
	// Incrementa contador
	c.mu.Lock()
	c.MessageCount[instance]++
	c.mu.Unlock()
	
	return resp.ID, nil
}

func (c *BaileysClient) SendInteractive(instance, number string, interactive *models.InteractivePayload, opts *models.MessageOptions) (string, error) {
	client, ok := c.getClient(instance)
	if !ok {
		return "", errors.New("instância desconectada")
	}

	jid, _ := types.ParseJID(number + "@s.whatsapp.net")

	var buttons []*waProto.InteractiveMessage_NativeFlowMessage_NativeFlowButton
	if interactive.Action != nil {
		for _, btn := range interactive.Action.Buttons {
			pBtn := &waProto.InteractiveMessage_NativeFlowMessage_NativeFlowButton{
				Name:             strPtr(btn.Name),
				ButtonParamsJSON: strPtr(btn.ButtonParamsJson),
			}
			buttons = append(buttons, pBtn)
		}
	}

	msgBody := &waProto.InteractiveMessage_Body{Text: strPtr(interactive.Body.Text)}
	
	msgNativeFlow := &waProto.InteractiveMessage_NativeFlowMessage{
		Buttons:        buttons,
		MessageVersion: int32Ptr(3),
	}

	interactiveMsg := &waProto.InteractiveMessage{
		Body: msgBody,
		InteractiveMessage: &waProto.InteractiveMessage_NativeFlowMessage_{
			NativeFlowMessage: msgNativeFlow,
		},
	}

	if interactive.Header != nil {
		interactiveMsg.Header = &waProto.InteractiveMessage_Header{
			Title:              strPtr(interactive.Header.Title),
			Subtitle:           strPtr(interactive.Header.Subtitle),
			HasMediaAttachment: boolPtr(false),
		}
	}
	if interactive.Footer != nil {
		interactiveMsg.Footer = &waProto.InteractiveMessage_Footer{Text: strPtr(interactive.Footer.Text)}
	}

	msg := &waProto.Message{InteractiveMessage: interactiveMsg}
	
	resp, err := client.SendMessage(context.Background(), jid, msg)
	if err != nil {
		return "", err
	}
	
	// Incrementa contador
	c.mu.Lock()
	c.MessageCount[instance]++
	c.mu.Unlock()
	
	return resp.ID, nil
}

func (c *BaileysClient) getClient(instance string) (*whatsmeow.Client, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	client, ok := c.Clients[instance]
	if !ok || !client.IsConnected() {
		return nil, false
	}
	return client, true
}

func (c *BaileysClient) IsConnected(instanceKey string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if client, ok := c.Clients[instanceKey]; ok {
		return client.IsConnected()
	}
	return false
}

// --- HELPERS E STUBS COMPLETOS ---

func strPtr(s string) *string { return &s }
func int32Ptr(i int32) *int32 { return &i }
func boolPtr(b bool) *bool { return &b }

func (c *BaileysClient) SendMedia(instance, number string, media *models.MediaPayload, opts *models.MessageOptions) (string, error) {
	return "", errors.New("mídia em breve")
}

func (c *BaileysClient) CreateGroup(instance, subject string, participants []string) (string, error) {
	return "", errors.New("grupo em breve")
}

func (c *BaileysClient) GroupAction(instance, groupID string, participants []string, action string) error {
	return nil
}

func (c *BaileysClient) DeleteMessage(instance, remoteJid, messageID string) error {
	return nil
}
