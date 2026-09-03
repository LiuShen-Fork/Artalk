package entity

type CookedUser struct {
	ID            uint   `json:"id"`
	Name          string `json:"name"`
	Email         string `json:"email"`
	Link          string `json:"link"`
	BadgeName     string `json:"badge_name"`
	BadgeColor    string `json:"badge_color"`
	IsAdmin       bool   `json:"is_admin"`
	IsAIAssistant bool   `json:"is_ai_assistant,omitempty"`
	ReceiveEmail  bool   `json:"receive_email"`
}
