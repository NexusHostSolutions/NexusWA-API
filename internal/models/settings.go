package models

type InstanceSettings struct {
	RejectCalls  bool `json:"reject_calls"`
	IgnoreGroups bool `json:"ignore_groups"`
	AlwaysOnline bool `json:"always_online"`
	ReadMessages bool `json:"read_messages"`
	SyncHistory  bool `json:"sync_history"`
	ReadStatus   bool `json:"read_status"`
}

type EventsConfig struct {
	Webhook   WebhookConfig  `json:"webhook"`
	RabbitMQ  RabbitMQConfig `json:"rabbitmq"`
	SQS       SQSConfig      `json:"sqs"`
	WebSocket bool           `json:"websocket_enabled"`
}

type WebhookConfig struct {
	Enabled bool   `json:"enabled"`
	URL     string `json:"url"`
}

type RabbitMQConfig struct {
	Enabled bool   `json:"enabled"`
	URI     string `json:"uri"`
	Queue   string `json:"queue"`
}

type SQSConfig struct {
	Enabled   bool   `json:"enabled"`
	Region    string `json:"region"`
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
}

type IntegrationsConfig struct {
	Typebot  TypebotConfig  `json:"typebot"`
	Chatwoot ChatwootConfig `json:"chatwoot"`
	OpenAI   OpenAIConfig   `json:"openai"`
	Dify     DifyConfig     `json:"dify"`
}

type TypebotConfig struct {
	Enabled bool   `json:"enabled"`
	URL     string `json:"url"`
	Typebot string `json:"typebot_name"`
}

type ChatwootConfig struct {
	Enabled   bool   `json:"enabled"`
	AccountID string `json:"account_id"`
	Token     string `json:"token"`
	URL       string `json:"url"`
}

type OpenAIConfig struct {
	Enabled bool   `json:"enabled"`
	ApiKey  string `json:"api_key"`
	Model   string `json:"model"`
}

type DifyConfig struct {
	Enabled bool   `json:"enabled"`
	ApiKey  string `json:"api_key"`
	URL     string `json:"url"`
}