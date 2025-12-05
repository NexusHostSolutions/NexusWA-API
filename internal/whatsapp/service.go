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

func (s *Service) GetMessages(instanceKey, jid string) ([]map[string]interface{}, error) {
	return s.Client.GetMessages(instanceKey, jid)
}

// 🔥 Botões nativos
func (s *Service) SendButtons(instanceKey, number, message, footer, title string, buttons []map[string]string) (string, error) {
	if !s.Client.IsConnected(instanceKey) {
		return "", errors.New("instance not connected")
	}
	return s.Client.SendButtons(instanceKey, number, message, footer, title, buttons)
}

// 🔥 Lista de seleção
func (s *Service) SendList(instanceKey, number, title, message, footer, buttonText string, sections []map[string]interface{}) (string, error) {
	if !s.Client.IsConnected(instanceKey) {
		return "", errors.New("instance not connected")
	}
	return s.Client.SendList(instanceKey, number, title, message, footer, buttonText, sections)
}

// 🔥 Botão com URL
func (s *Service) SendUrlButton(instanceKey, number, message, footer, title, buttonText, url string) (string, error) {
	if !s.Client.IsConnected(instanceKey) {
		return "", errors.New("instance not connected")
	}
	return s.Client.SendUrlButton(instanceKey, number, message, footer, title, buttonText, url)
}

// 🔥 Botão de copiar
func (s *Service) SendCopyButton(instanceKey, number, message, footer, title, buttonText, copyCode string) (string, error) {
	if !s.Client.IsConnected(instanceKey) {
		return "", errors.New("instance not connected")
	}
	return s.Client.SendCopyButton(instanceKey, number, message, footer, title, buttonText, copyCode)
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

func (s *Service) GetEventBus() interface{} {
	return nil
}