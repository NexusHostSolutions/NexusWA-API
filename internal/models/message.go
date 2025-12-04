package models

type SendMessageRequest struct {
	Number      string              `json:"number"`
	Type        string              `json:"type"`
	Text        string              `json:"text,omitempty"`
	Media       *MediaPayload       `json:"media,omitempty"`
	Interactive *InteractivePayload `json:"interactive,omitempty"`
	Options     *MessageOptions     `json:"options,omitempty"`
}

type MediaPayload struct {
	URL      string `json:"url"`
	Caption  string `json:"caption,omitempty"`
	FileName string `json:"filename,omitempty"`
	Type     string `json:"type"`
}

type MessageOptions struct {
	ReplyTo   string `json:"reply_to,omitempty"`
	IsForward bool   `json:"is_forward,omitempty"`
	Delay     int    `json:"delay,omitempty"`
}

type InteractivePayload struct {
	Type   string   `json:"type"`
	Header *Header  `json:"header,omitempty"`
	Body   *Body    `json:"body"`
	Footer *Footer  `json:"footer,omitempty"`
	Action *Action  `json:"action"`
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
	Button   string         `json:"button,omitempty"`
	Buttons  []NativeButton `json:"buttons,omitempty"`
	Sections []ListSection  `json:"sections,omitempty"`
}

type NativeButton struct {
	Name             string `json:"name"`
	ButtonParamsJson string `json:"buttonParamsJson"`
}

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