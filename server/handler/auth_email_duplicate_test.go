package handler_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/artalkjs/artalk/v2/internal/entity"
	"github.com/artalkjs/artalk/v2/server/handler"
	"github.com/artalkjs/artalk/v2/server/middleware/limiter"
	"github.com/stretchr/testify/require"
)

// A guest profile and a registered account can share one email (the guest used
// one nickname, then registered under another). Password login must select the
// row that owns the password, not whichever row happens to come first.
func TestEmailPasswordLoginSelectsThePasswordOwner(t *testing.T) {
	app, router := NewApiTestApp()
	t.Cleanup(func() { require.NoError(t, app.Cleanup()) })
	app.Conf().Auth.Email.Enabled = true
	app.Conf().Captcha.Enabled = false
	app.Conf().LoginTimeout = 3600
	router.Use(limiter.ActionLimitMiddleware(app.App, limiter.ActionLimitConf{}))
	handler.AuthEmailRegister(app.App, router)
	handler.AuthEmailLogin(app.App, router)

	const email = "duplicate@example.org"
	guest, err := app.Dao().FindCreateUser("Guest nickname", email, "")
	require.NoError(t, err)
	require.Zero(t, guest.Password)

	require.NoError(t, app.Dao().DB().Create(&entity.UserEmailVerify{
		Email: email, Code: "112233", ExpiresAt: time.Now().Add(5 * time.Minute),
	}).Error)
	registered := handler.RequestAuthEmailRegister{
		Name: "Registered name", Email: email, Password: "correct-password", Code: "112233",
	}
	var created handler.ResponseUserLogin
	require.NoError(t, json.Unmarshal(
		flowRequest(t, router, http.MethodPost, "/auth/email/register", registered, "", http.StatusOK), &created))
	require.NotEqual(t, guest.ID, created.User.ID)

	var signedIn handler.ResponseUserLogin
	require.NoError(t, json.Unmarshal(flowRequest(t, router, http.MethodPost, "/auth/email/login",
		handler.RequestAuthEmailLogin{Email: email, Password: registered.Password}, "", http.StatusOK), &signedIn))
	require.Equal(t, created.User.ID, signedIn.User.ID)

	// A code proves the email, so the session must land on the account that can
	// actually sign in rather than the older guest profile.
	require.NoError(t, app.Dao().DB().Create(&entity.UserEmailVerify{
		Email: email, Code: "445566", ExpiresAt: time.Now().Add(5 * time.Minute),
	}).Error)
	var byCode handler.ResponseUserLogin
	require.NoError(t, json.Unmarshal(flowRequest(t, router, http.MethodPost, "/auth/email/login",
		handler.RequestAuthEmailLogin{Email: email, Code: "445566"}, "", http.StatusOK), &byCode))
	require.Equal(t, created.User.ID, byCode.User.ID)

	// "Forgot password" (empty name) resets the credentialed account; a reset
	// written onto the guest row would leave the account's password unchanged.
	require.NoError(t, app.Dao().DB().Create(&entity.UserEmailVerify{
		Email: email, Code: "778899", ExpiresAt: time.Now().Add(5 * time.Minute),
	}).Error)
	var reset handler.ResponseUserLogin
	require.NoError(t, json.Unmarshal(flowRequest(t, router, http.MethodPost, "/auth/email/register",
		handler.RequestAuthEmailRegister{Email: email, Password: "reset-password", Code: "778899"}, "", http.StatusOK), &reset))
	require.Equal(t, created.User.ID, reset.User.ID)
	signedUp := app.Dao().FindUserByID(created.User.ID)
	require.True(t, signedUp.CheckPassword("reset-password"))
	stillGuest := app.Dao().FindUserByID(guest.ID)
	require.False(t, stillGuest.CheckPassword("reset-password"))
}
