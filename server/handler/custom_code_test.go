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

type customCodeBody struct {
	CSS string `json:"css"`
	JS  string `json:"js"`
}

func customCodeTestApp(t *testing.T) (router *fiber.App, adminToken, userToken, confFile string) {
	t.Helper()

	confFile = copyExampleConfig(t)
	app, router := NewApiTestApp()
	t.Cleanup(func() { _ = app.Cleanup() })

	conf, err := config.NewFromFile(confFile)
	require.NoError(t, err)
	app.SetConf(conf)

	handler.CustomCodeGet(app.App, router)
	handler.CustomCodeSave(app.App, router)

	adminToken, err = common.LoginGetUserToken(app.Dao().FindUserByID(1000), app.Conf().AppKey, 60)
	require.NoError(t, err)
	userToken, err = common.LoginGetUserToken(app.Dao().FindUserByID(1001), app.Conf().AppKey, 60)
	require.NoError(t, err)

	return router, adminToken, userToken, confFile
}

func TestCustomCodeRoundTrip(t *testing.T) {
	router, adminToken, _, confFile := customCodeTestApp(t)

	css := ".atk-sidebar { --color-accent: #000; }"
	js := "console.info('console test js')"

	resp, err := securityRequest(router, http.MethodPut, "/sidebar/custom", map[string]any{
		"css": css,
		"js":  js,
	}, adminToken, "203.0.113.10")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var body customCodeBody
	readJSON(t, resp, &body)
	assert.Equal(t, css, body.CSS)
	assert.Equal(t, js, body.JS)

	// The snippets are public: the console fetches them before anyone logs in.
	resp, err = securityRequest(router, http.MethodGet, "/sidebar/custom", nil, "", "203.0.113.10")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	readJSON(t, resp, &body)
	assert.Equal(t, css, body.CSS)
	assert.Equal(t, js, body.JS)

	// The config file keeps its comments and gains the new section.
	raw, err := os.ReadFile(confFile)
	require.NoError(t, err)
	assert.Contains(t, string(raw), "console test js")
	assert.Contains(t, string(raw), "custom:")
	assert.Contains(t, string(raw), "# Management console customization", "YAML comments survive the patch")

	// Saving identical content is a no-op instead of a rewrite.
	resp, err = securityRequest(router, http.MethodPut, "/sidebar/custom", map[string]any{"css": css}, adminToken, "203.0.113.10")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestCustomCodeRejectsOversizedSnippet(t *testing.T) {
	router, adminToken, _, _ := customCodeTestApp(t)

	resp, err := securityRequest(router, http.MethodPut, "/sidebar/custom", map[string]any{
		"css": strings.Repeat("a", config.MaxCustomCodeSize+1),
	}, adminToken, "203.0.113.10")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestCustomCodeWriteRequiresAdmin(t *testing.T) {
	router, _, userToken, _ := customCodeTestApp(t)

	resp, err := securityRequest(router, http.MethodPut, "/sidebar/custom", map[string]any{"css": "x"}, userToken, "203.0.113.10")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusForbidden, resp.StatusCode, "a normal user must not inject code into the console")
}
