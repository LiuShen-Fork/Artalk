package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/artalkjs/artalk/v2/internal/auth/generic"
	"github.com/markbates/goth"
)

type SocialUser struct {
	goth.User
	RemoteUID string
	Link      string
}

func GetSocialUser(u goth.User) SocialUser {
	var link string
	if u.Provider == "github" {
		if l, ok := u.RawData["blog"].(string); ok && l != "" {
			link = l
		} else if l, ok := u.RawData["html_url"].(string); ok && l != "" {
			link = l
		}
	}
	if u.Provider == generic.ProviderName {
		if l, ok := u.RawData[generic.RawDataUserLinkKey].(string); ok {
			link = l
		}
	}

	// Email patch
	if u.Provider == "steam" {
		// @see https://stackoverflow.com/questions/31571267/steam-get-users-email-address
		u.Email = u.UserID + "@steam.com"
	}
	if u.Email == "" {
		if u.Provider == generic.ProviderName {
			sum := sha256.Sum256([]byte(u.UserID))
			u.Email = "oauth-" + hex.EncodeToString(sum[:16]) + "@generic.invalid"
		} else {
			u.Email = u.UserID + "@" + u.Provider + ".com"
		}
	}

	// Name patch
	if u.Name == "" && u.Email != "" {
		// try extract name from email
		u.Name = strings.Split(u.Email, "@")[0]
	}

	return SocialUser{
		User:      u,
		RemoteUID: u.UserID,
		Link:      link,
	}
}
