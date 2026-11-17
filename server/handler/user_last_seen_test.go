package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/artalkjs/artalk/v2/internal/core"
	"github.com/artalkjs/artalk/v2/internal/entity"
	"github.com/artalkjs/artalk/v2/server/common"
	"github.com/artalkjs/artalk/v2/server/handler"
	"github.com/artalkjs/artalk/v2/server/middleware/limiter"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	lastSeenIP = "203.0.113.9"
	lastSeenUA = "LastSeenTest/1.0"
)

// The recent IP / UA shown in the admin panel used to be written only for
// anonymous commenters on a site without social login, so logged-in users kept
// the value from their first registration forever.
func TestCommentCreateRecordsLastSeen(t *testing.T) {
	setup := func(t *testing.T, anonymousAllowed bool) (*core.App, *fiber.App) {
		t.Helper()
		app, router := NewApiTestApp()
		app.Conf().Captcha.Enabled = false
		app.Conf().Auth.Enabled = true
		app.Conf().Auth.Anonymous = anonymousAllowed
		router.Use(limiter.ActionLimitMiddleware(app.App, limiter.ActionLimitConf{}))
		handler.CommentCreate(app.App, router)
		t.Cleanup(func() { _ = app.Cleanup() })
		return app.App, router
	}

	// forwardedIP is false to simulate a direct connection where the trusted
	// proxy header is absent. The connection address itself is covered by the
	// clientIP unit test: Fiber's test harness reports it as unspecified.
	post := func(t *testing.T, router *fiber.App, user entity.User, token string) int {
		t.Helper()
		body, err := json.Marshal(handler.ParamsCommentCreate{
			Name:     user.Name,
			Email:    user.Email,
			Content:  "Where did I come from?",
			SiteName: "Site A",
			PageKey:  "/last-seen",
		})
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/comments", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", lastSeenUA)
		// The test app runs with ProxyHeader = X-Forwarded-For and Fiber reads
		// the client IP from that header.
		req.Header.Set("X-Forwarded-For", lastSeenIP)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}

		resp, err := router.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		return resp.StatusCode
	}

	resetUser := func(t *testing.T, app *core.App, user entity.User) {
		t.Helper()
		require.NoError(t, app.Dao().DB().Model(&user).Updates(map[string]any{"last_ip": "", "last_ua": ""}).Error)
	}

	assertLastSeen := func(t *testing.T, app *core.App, userID uint) {
		t.Helper()
		var fresh entity.User
		require.NoError(t, app.Dao().DB().First(&fresh, userID).Error)
		assert.Equal(t, lastSeenIP, fresh.LastIP)
		assert.Equal(t, lastSeenUA, fresh.LastUA)
	}

	t.Run("logged-in commenter", func(t *testing.T) {
		app, router := setup(t, false)

		user := app.Dao().FindUserByID(1001)
		require.False(t, user.IsEmpty())
		resetUser(t, app, user)

		token, err := common.LoginGetUserToken(user, app.Conf().AppKey, 3600)
		require.NoError(t, err)

		require.Equal(t, 200, post(t, router, user, token))
		assertLastSeen(t, app, user.ID)
	})

	t.Run("anonymous commenter on a site with social login", func(t *testing.T) {
		app, router := setup(t, true)

		user := app.Dao().FindUserByID(1001)
		require.False(t, user.IsEmpty())
		resetUser(t, app, user)

		require.Equal(t, 200, post(t, router, user, ""))
		assertLastSeen(t, app, user.ID)
	})

}

// Logging in must also refresh the recent IP / UA, otherwise a user who only
// browses (and never comments) is never recorded.
func TestLoginRecordsLastSeen(t *testing.T) {
	app, router := NewApiTestApp()
	defer app.Cleanup()

	router.Use(limiter.ActionLimitMiddleware(app.App, limiter.ActionLimitConf{}))
	handler.UserLogin(app.App, router)

	admin := app.Dao().FindUserByID(1000)
	require.False(t, admin.IsEmpty())
	require.NoError(t, app.Dao().DB().Model(&admin).Updates(map[string]any{"last_ip": "", "last_ua": ""}).Error)

	body, err := json.Marshal(handler.ParamsUserLogin{Email: admin.Email, Password: "123456"})
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/user/access_token", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", lastSeenUA)
	req.Header.Set("X-Forwarded-For", lastSeenIP)

	resp, err := router.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var fresh entity.User
	require.NoError(t, app.Dao().DB().First(&fresh, admin.ID).Error)
	assert.Equal(t, lastSeenIP, fresh.LastIP)
	assert.Equal(t, lastSeenUA, fresh.LastUA)
}
