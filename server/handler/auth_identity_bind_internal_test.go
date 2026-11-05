package handler

import (
	"testing"
	"time"

	"github.com/artalkjs/artalk/v2/internal/auth"
	"github.com/artalkjs/artalk/v2/internal/entity"
	"github.com/artalkjs/artalk/v2/test"
	"github.com/golang-jwt/jwt/v5"
	"github.com/markbates/goth"
	"github.com/stretchr/testify/require"
)

func socialUser(provider, remoteUID, token string) auth.SocialUser {
	return auth.SocialUser{
		User: goth.User{
			Provider:    provider,
			UserID:      remoteUID,
			AccessToken: token,
			ExpiresAt:   time.Now().Add(time.Hour),
		},
		RemoteUID: remoteUID,
	}
}

func TestBindSocialIdentityDoesNotCreateUsersOrStealAccounts(t *testing.T) {
	app, err := test.NewTestApp()
	require.NoError(t, err)
	defer app.Cleanup()
	require.NoError(t, app.Dao().DB().Unscoped().Where("1 = 1").Delete(&entity.AuthIdentity{}).Error)

	var usersBefore int64
	require.NoError(t, app.Dao().DB().Model(&entity.User{}).Count(&usersBefore).Error)

	require.NoError(t, BindSocialIdentity(app.App, 1001, "github", socialUser("github", "remote-a", "token-a")))
	identity := app.Dao().FindAuthIdentityByUserID("github", 1001)
	require.EqualValues(t, 1001, identity.UserID)
	require.Equal(t, "remote-a", identity.RemoteUID)
	require.Equal(t, "token-a", identity.Token)

	// Re-binding the same provider account refreshes the stored token instead
	// of creating a duplicate row.
	require.NoError(t, BindSocialIdentity(app.App, 1001, "github", socialUser("github", "remote-a", "token-b")))
	var count int64
	require.NoError(t, app.Dao().DB().Model(&entity.AuthIdentity{}).
		Where("provider = ? AND user_id = ?", "github", 1001).Count(&count).Error)
	require.EqualValues(t, 1, count)
	require.Equal(t, "token-b", app.Dao().FindAuthIdentityByUserID("github", 1001).Token)

	// The same remote identity cannot be moved to another account.
	require.ErrorIs(t,
		BindSocialIdentity(app.App, 1002, "github", socialUser("github", "remote-a", "token-a")),
		ErrIdentityBoundToOtherUser,
	)

	// One account can only hold one identity per provider.
	require.ErrorIs(t,
		BindSocialIdentity(app.App, 1001, "github", socialUser("github", "remote-b", "token-b")),
		ErrProviderAlreadyBound,
	)

	// A different provider can still be linked.
	require.NoError(t, BindSocialIdentity(app.App, 1001, "gitlab", socialUser("gitlab", "remote-c", "token-c")))

	var usersAfter int64
	require.NoError(t, app.Dao().DB().Model(&entity.User{}).Count(&usersAfter).Error)
	require.Equal(t, usersBefore, usersAfter)

	// The database itself must refuse a duplicate remote identity, which is
	// the final guard against two concurrent bind requests.
	require.Error(t, app.Dao().DB().Create(&entity.AuthIdentity{
		Provider: "github", RemoteUID: "remote-a", UserID: 1002,
	}).Error)
}

func TestIdentityBindStateValidation(t *testing.T) {
	app, err := test.NewTestApp()
	require.NoError(t, err)
	defer app.Cleanup()
	require.NoError(t, app.Dao().DB().Unscoped().Where("1 = 1").Delete(&entity.AuthIdentity{}).Error)

	state, err := newIdentityBindState(app.App, 1001, "github")
	require.NoError(t, err)
	require.Contains(t, state, identityBindStatePrefix)

	claims, isBind, err := parseIdentityBindState(app.App, state)
	require.NoError(t, err)
	require.True(t, isBind)
	require.EqualValues(t, 1001, claims.UserID)
	require.Equal(t, "github", claims.Provider)

	// Tampering must never be accepted. Append rather than overwrite the last
	// base64 character, whose unused trailing bits can decode identically.
	_, isBind, err = parseIdentityBindState(app.App, state+"x")
	require.True(t, isBind)
	require.Error(t, err)

	// Ordinary OAuth login state is not a bind request.
	_, isBind, err = parseIdentityBindState(app.App, "random-oauth-state")
	require.NoError(t, err)
	require.False(t, isBind)

	// Expired bind states are rejected.
	expired := identityBindClaims{
		UserID:   1001,
		Provider: "github",
		Purpose:  identityBindPurpose,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-time.Hour)),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute)),
		},
	}
	raw, err := jwt.NewWithClaims(jwt.SigningMethodHS256, expired).SignedString([]byte(app.Conf().AppKey))
	require.NoError(t, err)
	_, isBind, err = parseIdentityBindState(app.App, identityBindStatePrefix+raw)
	require.True(t, isBind)
	require.Error(t, err)
}
