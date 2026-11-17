package security_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/artalkjs/artalk/v2/internal/config"
	"github.com/artalkjs/artalk/v2/internal/core"
	"github.com/artalkjs/artalk/v2/internal/entity"
	"github.com/artalkjs/artalk/v2/server/middleware/security"
	"github.com/artalkjs/artalk/v2/test"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const guardTestIP = "203.0.113.50"

type guardEnv struct {
	app     *test.TestApp
	fiberApp *fiber.App
	svc     *core.SecurityService
}

// newGuardEnv 构造一个启用了安全防护的测试应用
func newGuardEnv(t *testing.T, configure func(conf *config.Config)) *guardEnv {
	t.Helper()

	app, err := test.NewTestApp()
	require.NoError(t, err)
	t.Cleanup(func() { _ = app.Cleanup() })

	conf := app.Conf()
	conf.Security.Enabled = true
	conf.Security.Action = config.SecurityActionReject
	conf.Captcha.Enabled = true
	conf.Captcha.CaptchaType = config.TypeImage
	if configure != nil {
		configure(conf)
	}
	app.SetConf(conf)

	svc, err := core.AppService[*core.SecurityService](app.App)
	require.NoError(t, err)
	require.NotNil(t, svc)
	svc.ApplyConfig()

	fiberApp := fiber.New(fiber.Config{ProxyHeader: "X-Forwarded-For"})
	fiberApp.Use(security.Guard(app.App))
	fiberApp.Post("/api/v2/comments", func(c *fiber.Ctx) error { return c.SendString("ok") })
	fiberApp.Get("/api/v2/comments", func(c *fiber.Ctx) error { return c.SendString("ok") })

	return &guardEnv{app: app, fiberApp: fiberApp, svc: svc}
}

func (e *guardEnv) post(t *testing.T, ip string) (*http.Response, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v2/comments", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-For", ip)

	resp, err := e.fiberApp.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	data := map[string]any{}
	if trimmed := strings.TrimSpace(string(body)); strings.HasPrefix(trimmed, "{") {
		require.NoError(t, json.Unmarshal(body, &data))
	}
	return resp, data
}

func TestGuardPassesUntilQuotaThenRejects(t *testing.T) {
	env := newGuardEnv(t, func(conf *config.Config) {
		conf.Security.RateLimit.IP = config.SecurityQuotaConf{Minute: 2}
	})

	for i := 0; i < 2; i++ {
		resp, _ := env.post(t, guardTestIP)
		require.Equal(t, http.StatusOK, resp.StatusCode, "request #%d", i+1)
	}

	resp, data := env.post(t, guardTestIP)
	assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
	assert.Equal(t, true, data["rate_limited"])
	assert.NotEmpty(t, resp.Header.Get(fiber.HeaderRetryAfter))
	assert.NotZero(t, data["retry_after"])

	// 其他 IP 不受影响
	resp, _ = env.post(t, "203.0.113.51")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestGuardCaptchaAction(t *testing.T) {
	env := newGuardEnv(t, func(conf *config.Config) {
		conf.Security.Action = config.SecurityActionCaptcha
		conf.Security.RateLimit.IP = config.SecurityQuotaConf{Minute: 1}
	})

	resp, _ := env.post(t, guardTestIP)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	resp, data := env.post(t, guardTestIP)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	assert.Equal(t, true, data["need_captcha"])
	assert.NotEmpty(t, data["img_data"], "the image captcha payload keeps the official widget contract")
}

func TestGuardRejectsBannedIP(t *testing.T) {
	env := newGuardEnv(t, func(conf *config.Config) {
		conf.Security.RateLimit.IP = config.SecurityQuotaConf{}
	})

	_, err := env.svc.BanIP(guardTestIP, "test ban", "", -1, 0, entity.IPBanSourceManual)
	require.NoError(t, err)

	resp, data := env.post(t, guardTestIP)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	assert.Equal(t, true, data["ip_banned"])

	resp, _ = env.post(t, "203.0.113.51")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestGuardWhitelistBypassesEverything(t *testing.T) {
	env := newGuardEnv(t, func(conf *config.Config) {
		conf.Security.RateLimit.IP = config.SecurityQuotaConf{Minute: 1}
		conf.Security.Whitelist = []string{guardTestIP}
	})

	for i := 0; i < 5; i++ {
		resp, _ := env.post(t, guardTestIP)
		require.Equal(t, http.StatusOK, resp.StatusCode, "whitelisted ip should never be limited")
	}

	assert.Equal(t, 0, env.svc.Stats("").TrackedIPs, "whitelisted requests are not counted at all")
}

func TestGuardIgnoresReadRequests(t *testing.T) {
	env := newGuardEnv(t, func(conf *config.Config) {
		conf.Security.RateLimit.IP = config.SecurityQuotaConf{Minute: 1}
	})

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v2/comments", nil)
		req.Header.Set("X-Forwarded-For", guardTestIP)
		resp, err := env.fiberApp.Test(req, -1)
		require.NoError(t, err)
		resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
	}
}

func TestGuardDisabledPassesThrough(t *testing.T) {
	env := newGuardEnv(t, func(conf *config.Config) {
		conf.Security.Enabled = false
		conf.Security.RateLimit.IP = config.SecurityQuotaConf{Minute: 1}
	})

	for i := 0; i < 5; i++ {
		resp, _ := env.post(t, guardTestIP)
		require.Equal(t, http.StatusOK, resp.StatusCode)
	}
}

func TestGuardExcludedPath(t *testing.T) {
	env := newGuardEnv(t, func(conf *config.Config) {
		conf.Security.RateLimit.IP = config.SecurityQuotaConf{Minute: 1}
		conf.Security.ExcludePaths = []string{"/comments"}
	})

	for i := 0; i < 5; i++ {
		resp, _ := env.post(t, guardTestIP)
		require.Equal(t, http.StatusOK, resp.StatusCode)
	}
}
