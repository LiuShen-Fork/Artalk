package entity

import (
	"time"

	"gorm.io/gorm"
)

type AuthIdentity struct {
	gorm.Model
	Provider string `gorm:"uniqueIndex:idx_auth_identity_provider_remote_uid,priority:1;uniqueIndex:idx_auth_identity_provider_user,priority:1"` // local, email, oauth, github
	// A remote identity may only ever be attached to one Artalk account, and an
	// account may only have one identity per provider. The database enforces both
	// invariants so two concurrent binds cannot claim the same account.
	RemoteUID   string `gorm:"column:remote_uid;index;size:255;uniqueIndex:idx_auth_identity_provider_remote_uid,priority:2"`
	UserID      uint   `gorm:"index;uniqueIndex:idx_auth_identity_provider_user,priority:2"`
	Token       string
	ConfirmedAt *time.Time
	ExpiresAt   *time.Time
}

func (n AuthIdentity) IsEmpty() bool {
	return n.ID == 0
}
