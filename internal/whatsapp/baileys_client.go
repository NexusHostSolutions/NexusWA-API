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
	Clients   map[string]*whatsmeow.Client
	mu        sync.RWMutex
}

func NewBaileysClient() *BaileysClient {
	_ = os.Mkdir("sessions", 0755)
	return &BaileysClient{
		Clients: make(map[string]*whatsmeow.Client),
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

	// CORREÇÃO: Removida configuração manual de OS/Platform que causava erro
	// O whatsmeow gerencia a identidade internamente agora

	clientLog := waLog.Stdout("Client", "INFO", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)
	c.Clients[instanceKey] = client

	client.AddEventHandler(func(evt interface{}) {
		switch v := evt.(type) {
		case *events.Message:
			if !v.Info.IsFromMe {
				log.Printf("[MSG] %s: Recebida de %s", instanceKey, v.Info.Sender.User)
			}
		case *events.Connected:
			log.Printf("[%s] 🟢 Conectado!", instanceKey)
		}
	})

	if client.Store.ID != nil {
		err = client.Connect()
		if err != nil {
			return nil, err
		}
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
			}
		}
		close(qrStringChan)
	}()

	return qrStringChan, nil
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

	// Busca foto de perfil
	var avatarURL string
	picInfo, err := client.GetProfilePictureInfo(context.Background(), *client.Store.ID, &whatsmeow.GetProfilePictureParams{Preview: false})
	if err == nil && picInfo != nil {
		avatarURL = picInfo.URL
	}

	// Estatísticas de Contatos e Grupos
	contactsCount := 0
	groupsCount := 0
	
	contacts, err := client.Store.Contacts.GetAllContacts(context.Background())
	if err == nil {
		for jid := range contacts {
			if jid.Server == "g.us" {
				groupsCount++
			} else {
				contactsCount++
			}
		}
	}

	return map[string]interface{}{
		"jid":      jid,
		"name":     pushName,
		"avatar":   avatarURL,
		"status":   "connected",
		"contacts": contactsCount,
		"groups":   groupsCount,
	}, nil
}

// --- CHAT: LISTA DE CONTATOS ---

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

		result = append(result, map[string]interface{}{
			"jid":      jid.String(),
			"name":     name,
			"is_group": jid.Server == "g.us",
			"unread":   0, 
		})
	}
	return result, nil
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