package handler

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/artalkjs/artalk/v2/internal/auth"
	"github.com/artalkjs/artalk/v2/internal/core"
	"github.com/artalkjs/artalk/v2/internal/entity"
	"github.com/golang-jwt/jwt/v5"
)

const (
	identityBindStatePrefix = "atkbind."
	identityBindPurpose     = "identity_bind"
	identityBindTTL         = 10 * time.Minute
)

var (
	// ErrIdentityBoundToOtherUser means the remote account already belongs to
	// another Artalk account. It must never be silently transferred.
	ErrIdentityBoundToOtherUser = errors.New("this account is already bound to another user")
	// ErrProviderAlreadyBound means the Artalk account already has a different
	// identity for this provider. Binding a second one would make unlinking
	// ambiguous and is intentionally rejected.
	ErrProviderAlreadyBound = errors.New("this account already has a different binding for this provider")
)

type identityBindClaims struct {
	UserID   uint   `json:"user_id"`
	Provider string `json:"provider"`
	Purpose  string `json:"purpose"`
	Nonce    string `json:"nonce"`
	jwt.RegisteredClaims
}

func newIdentityBindState(app *core.App, userID uint, provider string) (string, error) {
	now := time.Now()
	claims := identityBindClaims{
		UserID:   userID,
		Provider: strings.ToLower(strings.TrimSpace(provider)),
		Purpose:  identityBindPurpose,
		Nonce:    fmt.Sprintf("%d-%d", userID, now.UnixNano()),
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(identityBindTTL)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(app.Conf().AppKey))
	if err != nil {
		return "", err
	}
	return identityBindStatePrefix + signed, nil
}

// parseIdentityBindState returns isBind=false for the ordinary OAuth login
// state. A malformed state that carries the bind prefix is still reported as a
// bind attempt so callers can show an error instead of falling back to login.
func parseIdentityBindState(app *core.App, state string) (claims identityBindClaims, isBind bool, err error) {
	state = strings.TrimSpace(state)
	if !strings.HasPrefix(state, identityBindStatePrefix) {
		return claims, false, nil
	}

	raw := strings.TrimPrefix(state, identityBindStatePrefix)
	token, err := jwt.ParseWithClaims(raw, &claims, func(token *jwt.Token) (any, error) {
		return []byte(app.Conf().AppKey), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired())
	if err != nil || token == nil || !token.Valid {
		return claims, true, errors.New("invalid or expired bind request")
	}
	if claims.Purpose != identityBindPurpose || claims.UserID == 0 || strings.TrimSpace(claims.Provider) == "" {
		return claims, true, errors.New("invalid bind request")
	}
	return claims, true, nil
}

func isEnabledOAuthProvider(app *core.App, provider string) bool {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "" || provider == "email" || provider == "skip" {
		return false
	}
	for _, p := range auth.GetProviders(app.Conf()) {
		if strings.EqualFold(p.Name(), provider) {
			return true
		}
	}
	return false
}

// BindSocialIdentity links a completed OAuth identity to an existing Artalk
// user. Unlike RegisterSocialUser it never creates or selects a user from the
// provider email, because an email claim is not proof of account ownership.
func BindSocialIdentity(app *core.App, userID uint, provider string, socialUser auth.SocialUser) error {
	if userID == 0 || strings.TrimSpace(provider) == "" || strings.TrimSpace(socialUser.RemoteUID) == "" {
		return errors.New("invalid identity")
	}

	now := time.Now()
	existing := app.Dao().FindAuthIdentityByRemoteUID(provider, socialUser.RemoteUID)
	if !existing.IsEmpty() {
		if existing.UserID != userID {
			return ErrIdentityBoundToOtherUser
		}
		return updateBoundIdentity(app, &existing, socialUser, now)
	}

	own := app.Dao().FindAuthIdentityByUserID(provider, userID)
	if !own.IsEmpty() {
		if own.RemoteUID != socialUser.RemoteUID {
			return ErrProviderAlreadyBound
		}
		return updateBoundIdentity(app, &own, socialUser, now)
	}

	identity := entity.AuthIdentity{
		Provider:    strings.ToLower(strings.TrimSpace(provider)),
		RemoteUID:   socialUser.RemoteUID,
		UserID:      userID,
		Token:       socialUser.AccessToken,
		ConfirmedAt: &now,
	}
	if !socialUser.ExpiresAt.IsZero() {
		identity.ExpiresAt = &socialUser.ExpiresAt
	}
	if err := app.Dao().CreateAuthIdentity(&identity); err != nil {
		// Resolve a race through the unique index without ever moving an
		// identity that belongs to someone else.
		again := app.Dao().FindAuthIdentityByRemoteUID(provider, socialUser.RemoteUID)
		if !again.IsEmpty() && again.UserID == userID {
			return nil
		}
		if !again.IsEmpty() {
			return ErrIdentityBoundToOtherUser
		}
		return err
	}
	return nil
}

func updateBoundIdentity(app *core.App, identity *entity.AuthIdentity, socialUser auth.SocialUser, now time.Time) error {
	identity.Token = socialUser.AccessToken
	identity.ConfirmedAt = &now
	if !socialUser.ExpiresAt.IsZero() {
		identity.ExpiresAt = &socialUser.ExpiresAt
	} else {
		identity.ExpiresAt = nil
	}
	return app.Dao().UpdateAuthIdentity(identity)
}
