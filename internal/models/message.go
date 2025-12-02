package models

// Estruturas genéricas de requisição
type SendMessageRequest struct {
	Number      string             `json:"number"`
	Type        string             `json:"type"` // text, media, button, list
	Text        string             `json:"text,omitempty"`
	Media       *MediaPayload      `json:"media,omitempty"`
	Interactive *InteractivePayload `json:"interactive,omitempty"`
	Options     *MessageOptions    `json:"options,omitempty"`
}

type MediaPayload struct {
	URL      string `json:"url"`
	Caption  string `json:"caption,omitempty"`
	FileName string `json:"filename,omitempty"`
	Type     string `json:"type"` // image, video, audio, document
}

type MessageOptions struct {
	ReplyTo    string `json:"reply_to,omitempty"`    // ID da mensagem para responder
	IsForward  bool   `json:"is_forward,omitempty"`  // Se é encaminhada
	Delay      int    `json:"delay,omitempty"`       // Delay simulado em segundos
}

// --- ESTRUTURAS OBRIGATÓRIAS 2025 (Native Flow) ---

type InteractivePayload struct {
	Type   string      `json:"type"` // native_flow, list, button
	Header *Header     `json:"header,omitempty"`
	Body   *Body       `json:"body"`
	Footer *Footer     `json:"footer,omitempty"`
	Action *Action     `json:"action"`
}

type Header struct {
	Title    string `json:"title,omitempty"`
	Subtitle string `json:"subtitle,omitempty"`
	HasMedia bool   `json:"has_media,omitempty"`
}

type Body struct {
	Text string `json:"text"`
}

type Footer struct {
	Text string `json:"text"`
}

type Action struct {
	Button   string         `json:"button,omitempty"`   // Texto do botão principal (para listas)
	Buttons  []NativeButton `json:"buttons,omitempty"`  // Para botões de ação
	Sections []ListSection  `json:"sections,omitempty"` // Para listas
}

type NativeButton struct {
	Name             string `json:"name"` // quick_reply, cta_url, etc
	ButtonParamsJson string `json:"buttonParamsJson"` // JSON stringified
}

// Helper para construir Params facilmente
type ButtonParams struct {
	ID          string `json:"id,omitempty"`
	DisplayText string `json:"display_text,omitempty"`
	URL         string `json:"url,omitempty"`
	CopyCode    string `json:"copy_code,omitempty"`
}

type ListSection struct {
	Title string    `json:"title"`
	Rows  []ListRow `json:"rows"`
}

type ListRow struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}