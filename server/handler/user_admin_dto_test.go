package handler_test

import (
	"testing"

	"github.com/artalkjs/artalk/v2/internal/entity"
	"github.com/stretchr/testify/require"
)

// The console tells guests from accounts by looking at has_password and
// identity_providers; both must reach the admin list unchanged.
func TestAdminUserDTOReportsLoginCredentialState(t *testing.T) {
	app, _ := NewApiTestApp()
	t.Cleanup(func() { require.NoError(t, app.Cleanup()) })

	user := app.Dao().FindUserByID(1001) // fixture commenter, no password
	require.Empty(t, user.Password)

	dto := app.Dao().BuildAdminUserDTO(&user)
	require.False(t, dto.HasPassword)
	require.Empty(t, dto.IdentityProviders)

	require.NoError(t, app.Dao().CreateAuthIdentity(&entity.AuthIdentity{
		UserID: user.ID, Provider: "github", RemoteUID: "dto-state-test",
	}))

	dto = app.Dao().BuildAdminUserDTO(&user)
	require.False(t, dto.HasPassword)
	require.Equal(t, []string{"github"}, dto.IdentityProviders, "an OAuth account is not a guest")

	admin := app.Dao().FindUserByID(1000)
	require.True(t, app.Dao().BuildAdminUserDTO(&admin).HasPassword)
}
