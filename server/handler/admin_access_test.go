package handler_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/artalkjs/artalk/v2/internal/core"
	"github.com/artalkjs/artalk/v2/server/common"
	"github.com/artalkjs/artalk/v2/server/handler"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
)

// adminOnlyRoutes registers the admin surface: the sidebar's data endpoints plus
// every mutation that changes other people's content. A logged-in user who is
// not an admin must never reach them, and neither must an anonymous visitor.
func adminOnlyRoutes(app *core.App, router fiber.Router) {
	handler.UserList(app, router)
	handler.UserCreate(app, router)
	handler.UserUpdate(app, router)
	handler.UserDelete(app, router)
	handler.PageList(app, router)
	handler.PageUpdate(app, router)
	handler.CommentDelete(app, router)
	handler.CommentUpdate(app, router)
	handler.SiteList(app, router)
	handler.SettingGet(app, router)
	handler.Dashboard(app, router)
	handler.ModerationLogList(app, router)
	handler.ModerationLogDelete(app, router)
	handler.ModerationLogClear(app, router)
	handler.AIAssistantLogList(app, router)
	handler.AIAssistantLogDelete(app, router)
	handler.AIAssistantLogClear(app, router)
	handler.CacheFlush(app, router)
	handler.TransferExport(app, router)
}

func TestAdminEndpointsRejectNonAdmin(t *testing.T) {
	app, router := NewApiTestApp()
	defer app.Cleanup()

	adminOnlyRoutes(app.App, router)

	userToken, err := common.LoginGetUserToken(app.Dao().FindUserByID(1001), app.Conf().AppKey, 60)
	require.NoError(t, err)
	adminToken, err := common.LoginGetUserToken(app.Dao().FindUserByID(1000), app.Conf().AppKey, 60)
	require.NoError(t, err)

	routes := []struct{ method, path string }{
		{"GET", "/users/admin"},
		{"POST", "/users"},
		{"PUT", "/users/1002"},
		{"DELETE", "/users/1002"},
		{"GET", "/pages"},
		{"PUT", "/pages/1000"},
		{"DELETE", "/comments/1000"},
		{"PUT", "/comments/1000"},
		{"GET", "/sites"},
		{"GET", "/settings"},
		{"GET", "/dashboard"},
		{"GET", "/moderation/logs"},
		{"DELETE", "/moderation/logs/1"},
		{"DELETE", "/moderation/logs"},
		{"GET", "/ai-assistant/logs"},
		{"DELETE", "/ai-assistant/logs/1"},
		{"DELETE", "/ai-assistant/logs"},
		{"POST", "/cache/flush"},
		{"GET", "/transfer/export"},
	}

	for _, route := range routes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			resp, err := securityRequest(router, route.method, route.path, map[string]any{}, userToken, "203.0.113.10")
			require.NoError(t, err)
			resp.Body.Close()
			require.Equal(t, http.StatusForbidden, resp.StatusCode, "a normal user must not reach an admin endpoint")

			resp, err = securityRequest(router, route.method, route.path, map[string]any{}, "", "203.0.113.10")
			require.NoError(t, err)
			resp.Body.Close()
			require.Equal(t, http.StatusForbidden, resp.StatusCode, "an anonymous visitor must not reach an admin endpoint")
		})
	}

	// The admin reaches the same endpoints, so the guard is not blanket-denying.
	for _, path := range []string{"/users/admin", "/moderation/logs", "/dashboard"} {
		resp, err := securityRequest(router, "GET", path, nil, adminToken, "203.0.113.10")
		require.NoError(t, err)
		resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode, "the admin must reach %s", path)
	}
}

func TestUserInfoHasNoAdminFields(t *testing.T) {
	app, router := NewApiTestApp()
	defer app.Cleanup()

	handler.UserInfo(app.App, router)

	token, err := common.LoginGetUserToken(app.Dao().FindUserByID(1001), app.Conf().AppKey, 60)
	require.NoError(t, err)

	resp, err := securityRequest(router, "GET", "/user", nil, token, "203.0.113.10")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var body struct {
		User map[string]any `json:"user"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.NotNil(t, body.User, "the response carries the current user")

	for _, field := range []string{"last_ip", "last_ua", "token_valid_from"} {
		_, leaked := body.User[field]
		require.False(t, leaked, "%s must never be sent to the account holder", field)
	}
	require.Equal(t, false, body.User["is_admin"], "userA is not an admin")
}
