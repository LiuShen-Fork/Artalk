package handler_test

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/artalkjs/artalk/v2/internal/entity"
	"github.com/artalkjs/artalk/v2/server/common"
	"github.com/artalkjs/artalk/v2/server/handler"
	atktest "github.com/artalkjs/artalk/v2/test"
	"github.com/stretchr/testify/require"
)

// The test database file is shared by tests in this package, so identities
// created by one test must not leak into the next one.
func clearTestIdentities(t *testing.T, app *atktest.TestApp) {
	t.Helper()
	require.NoError(t, app.Dao().DB().Unscoped().Where("1 = 1").Delete(&entity.AuthIdentity{}).Error)
}

func TestUserIdentitiesRequireOwnership(t *testing.T) {
	app, router := NewApiTestApp()
	defer app.Cleanup()
	clearTestIdentities(t, app)
	handler.UserIdentityList(app.App, router)
	handler.UserIdentityBind(app.App, router)
	handler.UserIdentityUnbind(app.App, router)

	tokenA, err := common.LoginGetUserToken(app.Dao().FindUserByID(1001), app.Conf().AppKey, 60)
	require.NoError(t, err)

	// No token: nothing is readable or changeable.
	resp, err := securityRequest(router, "GET", "/user/identities", nil, "", "")
	require.NoError(t, err)
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	resp.Body.Close()

	require.NoError(t, app.Dao().DB().Create(&entity.AuthIdentity{
		Provider: "github", RemoteUID: "remote-user-a", UserID: 1001, Token: "provider-token-secret",
	}).Error)
	require.NoError(t, app.Dao().DB().Create(&entity.AuthIdentity{
		Provider: "gitlab", RemoteUID: "remote-user-b", UserID: 1002, Token: "other-provider-token",
	}).Error)

	resp, err = securityRequest(router, "GET", "/user/identities", nil, tokenA, "")
	require.NoError(t, err)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Contains(t, string(body), "github")
	require.NotContains(t, string(body), "gitlab")
	require.NotContains(t, string(body), "provider-token-secret")
	require.NotContains(t, string(body), "remote-user-a")

	// A user cannot remove another user's identity.
	var foreign entity.AuthIdentity
	require.NoError(t, app.Dao().DB().Where("remote_uid = ?", "remote-user-b").First(&foreign).Error)
	resp, err = securityRequest(router, "DELETE", "/user/identities/"+strconv.FormatUint(uint64(foreign.ID), 10), nil, tokenA, "")
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusNotFound, resp.StatusCode)

	// Last login method protection: userA has no password and one identity.
	var own entity.AuthIdentity
	require.NoError(t, app.Dao().DB().Where("remote_uid = ?", "remote-user-a").First(&own).Error)
	resp, err = securityRequest(router, "DELETE", "/user/identities/"+strconv.FormatUint(uint64(own.ID), 10), nil, tokenA, "")
	require.NoError(t, err)
	var errResp map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
	resp.Body.Close()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	require.Contains(t, errResp["msg"], "last login method")

	// With a second method the first can be removed.
	require.NoError(t, app.Dao().DB().Create(&entity.AuthIdentity{
		Provider: "gitlab", RemoteUID: "second-method", UserID: 1001,
	}).Error)
	resp, err = securityRequest(router, "DELETE", "/user/identities/"+strconv.FormatUint(uint64(own.ID), 10), nil, tokenA, "")
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Removing the now-last identity must also be refused.
	var second entity.AuthIdentity
	require.NoError(t, app.Dao().DB().Where("remote_uid = ?", "second-method").First(&second).Error)
	resp, err = securityRequest(router, "DELETE", "/user/identities/"+strconv.FormatUint(uint64(second.ID), 10), nil, tokenA, "")
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// Confirm another user's account is untouched.
	var remaining int64
	require.NoError(t, app.Dao().DB().Model(&entity.AuthIdentity{}).Where("user_id = ?", 1002).Count(&remaining).Error)
	require.EqualValues(t, 1, remaining)
}

func TestUserIdentityBindRequiresConfiguredProvider(t *testing.T) {
	app, router := NewApiTestApp()
	defer app.Cleanup()
	handler.UserIdentityBind(app.App, router)

	token, err := common.LoginGetUserToken(app.Dao().FindUserByID(1001), app.Conf().AppKey, 60)
	require.NoError(t, err)

	app.Conf().Auth.Enabled = false
	resp, err := securityRequest(router, "POST", "/user/identities/generic/bind", nil, token, "")
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusNotFound, resp.StatusCode)

	app.Conf().Auth.Enabled = true
	app.Conf().Auth.Callback = "http://localhost:23366/api/v2/auth/{provider}/callback"
	app.Conf().Auth.Generic.Enabled = true
	app.Conf().Auth.Generic.ClientID = "client"
	app.Conf().Auth.Generic.ClientSecret = "secret"
	app.Conf().Auth.Generic.AuthorizeURL = "https://provider.example.com/authorize"
	app.Conf().Auth.Generic.TokenURL = "https://provider.example.com/token"
	app.Conf().Auth.Generic.UserInfoURL = "https://provider.example.com/userinfo"

	resp, err = securityRequest(router, "POST", "/user/identities/github/bind", nil, token, "")
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusNotFound, resp.StatusCode)

	resp, err = securityRequest(router, "POST", "/user/identities/generic/bind", nil, token, "")
	require.NoError(t, err)
	var result handler.ResponseUserIdentityBindURL
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Contains(t, result.Path, "/auth/generic?state=atkbind.")

	// A bind URL is only issuable by the account owner.
	resp, err = securityRequest(router, "POST", "/user/identities/generic/bind", nil, "", "")
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestUserPasswordUpdateSecurity(t *testing.T) {
	app, router := NewApiTestApp()
	defer app.Cleanup()
	clearTestIdentities(t, app)
	handler.UserPasswordUpdate(app.App, router)
	handler.UserIdentityList(app.App, router)
	handler.UserInfo(app.App, router)

	// The package shares one database file; restore credentials so later
	// tests keep seeing the fixture accounts in their original state.
	admin := app.Dao().FindUserByID(1000)
	require.NotEmpty(t, admin.Password)
	originalAdminPassword := admin.Password
	originalAdminValidFrom := admin.TokenValidFrom
	defer func() {
		admin.Password = originalAdminPassword
		admin.TokenValidFrom = originalAdminValidFrom
		require.NoError(t, app.Dao().UpdateUser(&admin))
	}()

	adminToken, err := common.LoginGetUserToken(admin, app.Conf().AppKey, 60)
	require.NoError(t, err)

	userA := app.Dao().FindUserByID(1001)
	originalUserAPassword := userA.Password
	originalUserAValidFrom := userA.TokenValidFrom
	defer func() {
		userA.Password = originalUserAPassword
		userA.TokenValidFrom = originalUserAValidFrom
		require.NoError(t, app.Dao().UpdateUser(&userA))
	}()

	// The current password is mandatory and must be correct.
	resp, err := securityRequest(router, "POST", "/user/password", map[string]string{
		"new_password": "a-new-password",
	}, adminToken, "")
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	resp, err = securityRequest(router, "POST", "/user/password", map[string]string{
		"current_password": "wrong", "new_password": "a-new-password",
	}, adminToken, "")
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	resp, err = securityRequest(router, "POST", "/user/password", map[string]string{
		"current_password": "123456", "new_password": "short",
	}, adminToken, "")
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	resp, err = securityRequest(router, "POST", "/user/password", map[string]string{
		"current_password": "123456", "new_password": "a-new-password",
	}, adminToken, "")
	require.NoError(t, err)
	var changed handler.ResponseUserPasswordUpdate
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&changed))
	resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.NotEmpty(t, changed.Token)
	require.True(t, changed.User.HasPassword)

	// The previous session is revoked; the returned one is valid.
	resp, err = securityRequest(router, "GET", "/user/identities", nil, adminToken, "")
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	resp, err = securityRequest(router, "GET", "/user/identities", nil, changed.Token, "")
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// A password-less account must prove email ownership with a code.
	userToken, err := common.LoginGetUserToken(userA, app.Conf().AppKey, 60)
	require.NoError(t, err)
	app.Conf().Auth.Email.Enabled = false
	resp, err = securityRequest(router, "POST", "/user/password", map[string]string{
		"new_password": "another-password",
	}, userToken, "")
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	app.Conf().Auth.Email.Enabled = true
	challenge := entity.UserEmailVerify{
		Email: app.Dao().FindUserByID(1001).Email, Code: "654321", ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	require.NoError(t, app.Dao().DB().Create(&challenge).Error)

	resp, err = securityRequest(router, "POST", "/user/password", map[string]string{
		"new_password": "another-password",
	}, userToken, "")
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	resp, err = securityRequest(router, "POST", "/user/password", map[string]string{
		"new_password": "another-password", "code": "654321",
	}, userToken, "")
	require.NoError(t, err)
	var userChanged handler.ResponseUserPasswordUpdate
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&userChanged))
	resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.True(t, userChanged.User.HasPassword)

	// The old token is no longer accepted.
	resp, err = securityRequest(router, "GET", "/user/identities", nil, userToken, "")
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	resp, err = securityRequest(router, "GET", "/user/identities", nil, userChanged.Token, "")
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
}
