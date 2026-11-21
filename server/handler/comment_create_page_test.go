package handler_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/artalkjs/artalk/v2/server/common"
	"github.com/artalkjs/artalk/v2/server/handler"
	"github.com/artalkjs/artalk/v2/server/middleware/limiter"
	"github.com/stretchr/testify/require"
)

func TestRejectedCommentDoesNotCreatePage(t *testing.T) {
	app, router := NewApiTestApp()
	defer app.Cleanup()
	app.Conf().Captcha.Enabled = false
	router.Use(limiter.ActionLimitMiddleware(app.App, limiter.ActionLimitConf{}))
	handler.CommentCreate(app.App, router)
	for _, tc := range []struct {
		name              string
		rid               uint
		token             string
		anonymousDisabled bool
		status            int
	}{
		{"invalid token", 0, "invalid", false, 401},
		{"missing parent", 999999, "", false, 404},
		{"wrong page", 1000, "", false, 400},
		{"anonymous disabled", 0, "", true, 401},
	} {
		t.Run(tc.name, func(t *testing.T) {
			key := "/rejected/" + tc.name
			app.Conf().Auth.Enabled = tc.anonymousDisabled
			app.Conf().Auth.Anonymous = !tc.anonymousDisabled
			body, err := json.Marshal(handler.ParamsCommentCreate{
				Name: "Visitor", Email: "visitor@example.com", Content: "Test", SiteName: "Site A", PageKey: key, Rid: tc.rid,
			})
			require.NoError(t, err)
			req := httptest.NewRequest("POST", "/comments", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			if tc.token != "" {
				req.Header.Set("Authorization", "Bearer "+tc.token)
			}
			resp, err := router.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			require.Equal(t, tc.status, resp.StatusCode)
			require.True(t, app.Dao().FindPage(key, "Site A").IsEmpty())
		})
	}
}

func TestAnonymousCannotPostAsAccountWithLoginCredentials(t *testing.T) {
	app, router := NewApiTestApp()
	defer app.Cleanup()
	app.Conf().Captcha.Enabled = false
	app.Conf().Auth.Enabled = true
	app.Conf().Auth.Anonymous = true
	router.Use(limiter.ActionLimitMiddleware(app.App, limiter.ActionLimitConf{}))
	handler.CommentCreate(app.App, router)

	// Give an existing fixture account a login credential.
	account := app.Dao().FindUserByID(1001)
	require.NoError(t, account.SetPasswordEncrypt("secret-123"))
	require.NoError(t, app.Dao().UpdateUser(&account))

	post := func(key, token string) (int, string) {
		body, err := json.Marshal(handler.ParamsCommentCreate{
			Name: account.Name, Email: account.Email, Content: "Test",
			SiteName: "Site A", PageKey: key,
		})
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/comments", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		resp, err := router.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		raw, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		return resp.StatusCode, string(raw)
	}

	anonymousKey := "/anonymous-impersonation"
	status, body := post(anonymousKey, "")
	require.Equal(t, 401, status)
	require.Contains(t, body, "registered account", "the reason must be explained")
	require.True(t, app.Dao().FindPage(anonymousKey, "Site A").IsEmpty())

	token, err := common.LoginGetUserToken(account, app.Conf().AppKey, 3600)
	require.NoError(t, err)
	status, _ = post("/authenticated-account", token)
	require.Equal(t, 200, status)

	// A display name that has never registered still works anonymously.
	visitorBody, err := json.Marshal(handler.ParamsCommentCreate{
		Name: "Visitor", Email: "visitor-new@example.com", Content: "Test",
		SiteName: "Site A", PageKey: "/anonymous-visitor",
	})
	require.NoError(t, err)
	req := httptest.NewRequest("POST", "/comments", bytes.NewReader(visitorBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err := router.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, 200, resp.StatusCode)

	// Keep the fixture back in its original state for other tests.
	account.Password = ""
	require.NoError(t, app.Dao().UpdateUser(&account))
}
