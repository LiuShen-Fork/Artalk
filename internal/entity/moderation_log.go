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
	ModerationLogActionReject  ModerationLogAction = "reject"
)

type ModerationLog struct {
	gorm.Model

	CommentID uint   `gorm:"index" json:"comment_id"`
	SiteName  string `gorm:"index;size:255" json:"site_name"`
	PageKey   string `gorm:"index;size:255" json:"page_key"`
	UserID    uint   `gorm:"index" json:"user_id"`
	UserName  string `gorm:"size:255" json:"user_name"`
	UserEmail string `gorm:"size:255" json:"user_email"`

	// Snapshots are retained for checks that happen before a comment is saved.
	CommentContent string `json:"comment_content"`

	Checker string `gorm:"index;size:64" json:"checker"`
	Status  string `gorm:"index;size:32" json:"status"`
	Action  string `gorm:"index;size:32" json:"action"`
	Message string `json:"message"`
}

type CookedModerationLog struct {
	ID             uint   `json:"id"`
	CommentID      uint   `json:"comment_id"`
	SiteName       string `json:"site_name"`
	PageKey        string `json:"page_key"`
	UserID         uint   `json:"user_id"`
	UserName       string `json:"user_name"`
	UserEmail      string `json:"user_email"`
	CommentContent string `json:"comment_content"`
	Checker        string `json:"checker"`
	Status         string `json:"status"`
	Action         string `json:"action"`
	Message        string `json:"message"`
	Date           string `json:"date"`
}

type AIAssistantLogStatus string

const (
	AIAssistantLogStatusSuccess AIAssistantLogStatus = "success"
	AIAssistantLogStatusError   AIAssistantLogStatus = "error"
	AIAssistantLogStatusRateLimited AIAssistantLogStatus = "rate_limited"
)

type AIAssistantLog struct {
	gorm.Model

	CommentID      uint   `gorm:"index" json:"comment_id"`
	ReplyCommentID uint   `gorm:"index" json:"reply_comment_id"`
	UserID         uint   `gorm:"index" json:"user_id"`
	SiteName       string `gorm:"index;size:255" json:"site_name"`
	PageKey        string `gorm:"index;size:255" json:"page_key"`
	PageURL        string `json:"page_url"`
	Trigger        string `gorm:"size:64" json:"trigger"`
	Status         string `gorm:"index;size:32" json:"status"`
	AIModel        string `gorm:"size:255" json:"model"`
	Response       string `json:"response"`
	Error          string `json:"error"`
}
