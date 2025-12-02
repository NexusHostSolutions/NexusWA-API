package whatsapp

import (
	"errors"
	"log"
	"regexp"
	"time"

	"github.com/nexus/gowhats/internal/models"
)

// Service define a lógica de alto nível
type Service struct {
	Client *BaileysClient
}

func NewService() *Service {
	return &Service{
		Client: NewBaileysClient(),
	}
}

// SendMessage roteia a mensagem para o método correto do cliente
func (s *Service) SendMessage(instanceKey string, req models.SendMessageRequest) (string, error) {
	if !s.Client.IsConnected(instanceKey) {
		return "", errors.New("instance not connected")
	}

	// Lógica de Delay
	if req.Options != nil && req.Options.Delay > 0 {
		time.Sleep(time.Duration(req.Options.Delay) * time.Second)
	}

	switch req.Type {
	case "text":
		return s.Client.SendText(instanceKey, req.Number, req.Text, req.Options)
	case "media":
		return s.Client.SendMedia(instanceKey, req.Number, req.Media, req.Options)
	case "interactive":
		return s.Client.SendInteractive(instanceKey, req.Number, req.Interactive, req.Options)
	default:
		return "", errors.New("unsupported message type")
	}
}

// AntiLink Logic
func (s *Service) CheckAntiLink(instanceKey, groupID, sender, messageContent string) bool {
	// Regex simples para detectar links
	linkRegex := regexp.MustCompile(`(http|https):\/\/[^\s]+`)
	if linkRegex.MatchString(messageContent) {
		log.Printf("[ANTILINK] Detectado link de %s no grupo %s", sender, groupID)
		
		// 1. Apagar mensagem
		_ = s.Client.DeleteMessage(instanceKey, groupID, "msg_id_placeholder")
		
		// 2. Banir usuário (lógica simulada)
		_ = s.Client.GroupAction(instanceKey, groupID, []string{sender}, "remove")
		
		return true
	}
	return false
}

// Gerenciamento de Grupos
func (s *Service) ManageGroup(instanceKey string, action string, req models.GroupActionRequest) error {
	return s.Client.GroupAction(instanceKey, req.GroupID, req.Participants, action)
}

func (s *Service) CreateGroup(instanceKey string, req models.CreateGroupRequest) (string, error) {
	return s.Client.CreateGroup(instanceKey, req.Subject, req.Participants)
}