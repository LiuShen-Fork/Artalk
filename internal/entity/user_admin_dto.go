package entity

type AdminUserDTO struct {
	UserDTO
	LastIP   string `json:"last_ip"`
	LastUA   string `json:"last_ua"`
	IsInConf bool   `json:"is_in_conf"`
	// CommentCount counts the comments owned by this user row.
	CommentCount int64 `json:"comment_count"`
	// IdentityProviders names the linked third-party identities ("github",
	// "generic", ...). Together with HasPassword it separates a registered
	// account from the profile row that anonymous commenting leaves behind:
	// a guest has neither a password nor an identity.
	IdentityProviders []string `json:"identity_providers"`
}
