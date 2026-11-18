package handler_test

import (
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/artalkjs/artalk/v2/internal/config"
	"github.com/artalkjs/artalk/v2/server/common"
	"github.com/artalkjs/artalk/v2/server/handler"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type consoleBody struct {
	Name        string `json:"name"`
	Icon        string `json:"icon"`
	Description string `json:"description"`
	CSS         string `json:"css"`
	JS          string `json:"js"`
}

func consoleTestApp(t *testing.T) (router *fiber.App, adminToken, userToken, confFile string) {
	t.Helper()

	confFile = copyExampleConfig(t)
	app, router := NewApiTestApp()
	t.Cleanup(func() { _ = app.Cleanup() })

	conf, err := config.NewFromFile(confFile)
	require.NoError(t, err)
	app.SetConf(conf)

	handler.ConsoleGet(app.App, router)
	handler.ConsoleSave(app.App, router)

	adminToken, err = common.LoginGetUserToken(app.Dao().FindUserByID(1000), app.Conf().AppKey, 60)
	require.NoError(t, err)
	userToken, err = common.LoginGetUserToken(app.Dao().FindUserByID(1001), app.Conf().AppKey, 60)
	require.NoError(t, err)

	return router, adminToken, userToken, confFile
}

func TestConsoleRoundTrip(t *testing.T) {
	router, adminToken, _, confFile := consoleTestApp(t)

	payload := map[string]any{
		"name":        "示例站点",
		"icon":        "https://example.com/logo.png",
		"description": "一个用来演示控制台的站点",
		"css":         ".atk-sidebar { --color-accent: #000; }",
		"js":          "console.info('console test js')",
	}

	resp, err := securityRequest(router, http.MethodPut, "/sidebar/console", payload, adminToken, "203.0.113.10")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var body consoleBody
	readJSON(t, resp, &body)
	assert.Equal(t, "示例站点", body.Name)
	assert.Equal(t, "https://example.com/logo.png", body.Icon)
	assert.Equal(t, "一个用来演示控制台的站点", body.Description)
	assert.Equal(t, payload["css"], body.CSS)
	assert.Equal(t, payload["js"], body.JS)

	// The branding is public: the console fetches it before anyone logs in.
	resp, err = securityRequest(router, http.MethodGet, "/sidebar/console", nil, "", "203.0.113.10")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	readJSON(t, resp, &body)
	assert.Equal(t, "示例站点", body.Name)
	assert.Equal(t, payload["css"], body.CSS)

	// The config file keeps its comments and gains the new section.
	raw, err := os.ReadFile(confFile)
	require.NoError(t, err)
	assert.Contains(t, string(raw), "console test js")
	assert.Contains(t, string(raw), "console:")
	assert.NotContains(t, string(raw), "custom:")
	assert.Contains(t, string(raw), "# Management console branding", "YAML comments survive the patch")

	// Saving identical content is a no-op instead of a rewrite.
	resp, err = securityRequest(router, http.MethodPut, "/sidebar/console", map[string]any{"css": payload["css"]}, adminToken, "203.0.113.10")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestConsoleRejectsBadInput(t *testing.T) {
	router, adminToken, _, _ := consoleTestApp(t)

	cases := []struct {
		name string
		body map[string]any
	}{
		{"oversized snippet", map[string]any{"css": strings.Repeat("a", config.MaxConsoleCodeSize+1)}},
		{"oversized name", map[string]any{"name": strings.Repeat("名", config.MaxConsoleNameRunes+1)}},
		{"oversized description", map[string]any{"description": strings.Repeat("述", config.MaxConsoleDescriptionRunes+1)}},
		{"script icon", map[string]any{"icon": "javascript:alert(1)"}},
		{"protocol relative icon", map[string]any{"icon": "//example.com/logo.png"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := securityRequest(router, http.MethodPut, "/sidebar/console", tc.body, adminToken, "203.0.113.10")
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})
	}
}

func TestConsoleAcceptsRelativeIcon(t *testing.T) {
	router, adminToken, _, _ := consoleTestApp(t)

	resp, err := securityRequest(router, http.MethodPut, "/sidebar/console", map[string]any{
		"icon": "/static/logo.svg",
	}, adminToken, "203.0.113.10")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestConsoleWriteRequiresAdmin(t *testing.T) {
	router, _, userToken, _ := consoleTestApp(t)

	resp, err := securityRequest(router, http.MethodPut, "/sidebar/console", map[string]any{"css": "x"}, userToken, "203.0.113.10")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusForbidden, resp.StatusCode, "a normal user must not restyle or rename the console")
}
