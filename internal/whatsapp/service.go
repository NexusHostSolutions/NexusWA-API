package whatsapp

import (
	"errors"
	"time"

	"github.com/nexus/gowhats/internal/models"
)

type Service struct {
	Client *BaileysClient
}

func NewService() *Service {
	return &Service{
		Client: NewBaileysClient(),
	}
}

func (s *Service) SendMessage(instanceKey string, req models.SendMessageRequest) (string, error) {
	if !s.Client.IsConnected(instanceKey) {
		return "", errors.New("instance not connected")
	}
	if req.Options != nil && req.Options.Delay > 0 {
		time.Sleep(time.Duration(req.Options.Delay) * time.Second)
	}
	switch req.Type {
	case "text":
		return s.Client.SendText(instanceKey, req.Number, req.Text, req.Options)
	case "interactive":
		return s.Client.SendInteractive(instanceKey, req.Number, req.Interactive, req.Options)
	case "media":
		return s.Client.SendMedia(instanceKey, req.Number, req.Media, req.Options)
	default:
		return "", errors.New("unsupported message type")
	}
}

func (s *Service) GetInstanceInfo(instanceKey string) (map[string]interface{}, error) {
	return s.Client.GetConnectionInfo(instanceKey)
}

func (s *Service) GetContacts(instanceKey string) ([]map[string]interface{}, error) {
	return s.Client.GetContacts(instanceKey)
}

func (s *Service) GetGroups(instanceKey string) ([]map[string]interface{}, error) {
	return s.Client.GetGroups(instanceKey)
}

func (s *Service) SearchContacts(instanceKey, query string) ([]map[string]interface{}, error) {
	return s.Client.SearchContacts(instanceKey, query)
}

// 🔥 NOVO: Buscar mensagens de um chat
func (s *Service) GetMessages(instanceKey, jid string) ([]map[string]interface{}, error) {
	return s.Client.GetMessages(instanceKey, jid)
}

// Funções placeholder mantidas para compatibilidade com handlers
func (s *Service) PairPhone(instanceKey, phone string) (string, error) {
	return s.Client.PairPhone(instanceKey, phone)
}

func (s *Service) ManageGroup(instanceKey string, action string, req models.GroupActionRequest) error {
	return s.Client.GroupAction(instanceKey, req.GroupID, req.Participants, action)
}

func (s *Service) CreateGroup(instanceKey string, req models.CreateGroupRequest) (string, error) {
	return s.Client.CreateGroup(instanceKey, req.Subject, req.Participants)
}

// EventBus removido pois agora o Node.js gerencia eventos internamente
// Se precisar de Webhooks, o Node.js deve disparar o POST para seu servidor, 
// ou o Go pode fazer polling (não recomendado).
func (s *Service) GetEventBus() interface{} {
	return nil
}