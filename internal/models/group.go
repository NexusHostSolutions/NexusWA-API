package models

type CreateGroupRequest struct {
	Subject      string   `json:"subject"`
	Participants []string `json:"participants"` // Array de números
	Description  string   `json:"description,omitempty"`
}

type GroupActionRequest struct {
	GroupID      string   `json:"group_id"`
	Participants []string `json:"participants,omitempty"`
	Action       string   `json:"action"` // add, remove, promote, demote
}

type GroupInfo struct {
	ID           string   `json:"id"`
	Subject      string   `json:"subject"`
	Owner        string   `json:"owner"`
	Participants []string `json:"participants"`
	Admins       []string `json:"admins"`
	Creation     int64    `json:"creation"`
}