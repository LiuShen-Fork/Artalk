package auth

import (
	"fmt"
	"time"

	"github.com/artalkjs/artalk/v2/internal/dao"
	"github.com/artalkjs/artalk/v2/internal/entity"
	"github.com/artalkjs/artalk/v2/internal/utils"
)

func RegisterSocialUser(dao *dao.Dao, u SocialUser) (entity.AuthIdentity, error) {
	if u.Name == "" {
		return entity.AuthIdentity{}, fmt.Errorf("cannot fetch user name from social identity provider")
	}
	if u.Email == "" {
		return entity.AuthIdentity{}, fmt.Errorf("cannot fetch user email from social identity provider")
	}
	if !utils.ValidateEmail(u.Email) {
		return entity.AuthIdentity{}, fmt.Errorf("email is invalid which fetched from social identity provider")
	}

	var user entity.User
	var err error
	if u.Provider == "gitea" && u.RemoteUID != u.UserID {
		// A self-hosted server's profile is not proof of ownership of an existing
		// Artalk account, even when its name and email match.
		user, err = dao.NewUser(u.Name, u.Email, u.Link)
	} else {
		user, err = dao.FindCreateUser(u.Name, u.Email, u.Link)
	}
	if err != nil {
		return entity.AuthIdentity{}, err
	}

	var expiresAt *time.Time
	if !u.ExpiresAt.IsZero() {
		expiresAt = &u.ExpiresAt
	}

	// Store user auth identity in db
	now := time.Now()
	authIdentity := entity.AuthIdentity{
		Provider:    u.Provider,
		RemoteUID:   u.RemoteUID,
		UserID:      user.ID,
		Token:       u.AccessToken,
		ConfirmedAt: &now,
		ExpiresAt:   expiresAt,
	}
	if err := dao.CreateAuthIdentity(&authIdentity); err != nil {
		// The unique indexes make a concurrent login or a provider-side account
		// change observable here. Reuse the existing link instead of failing the
		// login; the account was already selected by the same name/email rules.
		if existing := dao.FindAuthIdentityByRemoteUID(u.Provider, u.RemoteUID); !existing.IsEmpty() {
			return existing, nil
		}
		if existing := dao.FindAuthIdentityByUserID(u.Provider, user.ID); !existing.IsEmpty() {
			return existing, nil
		}
		return entity.AuthIdentity{}, err
	}

	return authIdentity, nil
}
