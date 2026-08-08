package entity

import "gorm.io/gorm"

type ModerationLogStatus string

const (
	ModerationLogStatusPass  ModerationLogStatus = "pass"
	ModerationLogStatusBlock ModerationLogStatus = "block"
	ModerationLogStatusError ModerationLogStatus = "error"
)

type ModerationLogAction string

const (
	ModerationLogActionAllow   ModerationLogAction = "allow"
	ModerationLogActionPending ModerationLogAction = "pending"
	ModerationLogActionReplace ModerationLogAction = "replace"
)

type ModerationLog struct {
	gorm.Model

	CommentID uint   `gorm:"index" json:"comment_id"`
	SiteName  string `gorm:"index;size:255" json:"site_name"`
	PageKey   string `gorm:"index;size:255" json:"page_key"`
	UserID    uint   `gorm:"index" json:"user_id"`

	Checker string `gorm:"index;size:64" json:"checker"`
	Status  string `gorm:"index;size:32" json:"status"`
	Action  string `gorm:"index;size:32" json:"action"`
	Message string `json:"message"`
}

type CookedModerationLog struct {
	ID        uint   `json:"id"`
	CommentID uint   `json:"comment_id"`
	SiteName  string `json:"site_name"`
	PageKey   string `json:"page_key"`
	UserID    uint   `json:"user_id"`
	Checker   string `json:"checker"`
	Status    string `json:"status"`
	Action    string `json:"action"`
	Message   string `json:"message"`
	Date      string `json:"date"`
}
