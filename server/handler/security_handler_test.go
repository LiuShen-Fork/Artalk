package handler_test

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/artalkjs/artalk/v2/internal/config"
	"github.com/artalkjs/artalk/v2/internal/entity"
	"github.com/artalkjs/artalk/v2/server/common"
	"github.com/artalkjs/artalk/v2/server/handler"
	"github.com/artalkjs/artalk/v2/test"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func readJSON(t *testing.T, resp *http.Response, out any) {
	t.Helper()
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(body, out), "body: %s", string(body))
}

// securityTestApp 注册安全相关路由，返回应用、管理员 token
func securityTestApp(t *testing.T, confFile string) (*fiber.App, *test.TestApp, string) {
	t.Helper()

	testApp, router := NewApiTestApp()
	t.Cleanup(func() { _ = testApp.Cleanup() })

	if confFile != "" {
		conf, err := config.NewFromFile(confFile)
		require.NoError(t, err)
		testApp.SetConf(conf)
	}

	handler.SecurityConfigGet(testApp.App, router)
	handler.SecurityConfigSave(testApp.App, router)
	handler.SecurityEventList(testApp.App, router)
	handler.SecurityEventDelete(testApp.App, router)
	handler.SecurityEventClear(testApp.App, router)
	handler.SecurityReport(testApp.App, router)
	handler.SecurityBanList(testApp.App, router)
	handler.SecurityBanCreate(testApp.App, router)
	handler.SecurityBanDelete(testApp.App, router)
	handler.SecurityStatsGet(testApp.App, router)

	token, err := common.LoginGetUserToken(testApp.Dao().FindUserByID(1000), testApp.Conf().AppKey, 60)
	require.NoError(t, err)

	return router, testApp, token
}

func copyExampleConfig(t *testing.T) string {
	t.Helper()

	// test.NewTestApp() 会把工作目录切到项目根目录，这里两种相对路径都尝试
	var (
		src []byte
		err error
	)
	for _, path := range []string{"../../conf/artalk.example.yml", "conf/artalk.example.yml"} {
		if src, err = os.ReadFile(path); err == nil {
			break
		}
	}
	require.NoError(t, err)

	target := filepath.Join(t.TempDir(), "artalk.yml")
	require.NoError(t, os.WriteFile(target, src, 0o600))
	return target
}

func TestSecurityConfigSavePatchesYAML(t *testing.T) {
	confFile := copyExampleConfig(t)
	router, _, token := securityTestApp(t, confFile)

	resp, err := securityRequest(router, http.MethodPut, "/security/config", map[string]any{
		"enabled": true,
		"action":  "reject",
		"rate_limit": map[string]any{
			"ip":     map[string]any{"minute": 5, "hour": 50, "day": 500},
			"global": map[string]any{"minute": 30, "hour": 0, "day": 0},
		},
		"auto_ban":      map[string]any{"enabled": true, "threshold": 3, "window": 600, "duration": 3600},
		"blacklist":     []string{"203.0.113.9", "10.0.0.0/8"},
		"exclude_paths": []string{"/captcha/verify", "/webhook/"},
		"whitelist":     []string{"127.0.0.1"},
		"captcha": map[string]any{
			"enabled":      true,
			"captcha_type": "geetest",
			"geetest":      map[string]any{"captcha_id": "gt-id", "captcha_key": "gt-key"},
			"turnstile":    map[string]any{"site_key": "ts-site", "secret_key": "ts-secret"},
		},
	}, token, "203.0.113.10")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var body struct {
		Security struct {
			Enabled   bool   `json:"enabled"`
			Action    string `json:"action"`
			RateLimit struct {
				IP     config.SecurityQuotaConf `json:"ip"`
				Global config.SecurityQuotaConf `json:"global"`
			} `json:"rate_limit"`
			AutoBan struct {
				Enabled   bool `json:"enabled"`
				Threshold int  `json:"threshold"`
			} `json:"auto_ban"`
			Blacklist    []string `json:"blacklist"`
			Whitelist    []string `json:"whitelist"`
			ExcludePaths []string `json:"exclude_paths"`
		} `json:"security"`
		Captcha struct {
			CaptchaType string `json:"captcha_type"`
			Geetest     struct {
				CaptchaID  string `json:"captcha_id"`
				CaptchaKey string `json:"captcha_key"`
			} `json:"geetest"`
			Turnstile struct {
				SiteKey   string `json:"site_key"`
				SecretKey string `json:"secret_key"`
			} `json:"turnstile"`
		} `json:"captcha"`
	}
	readJSON(t, resp, &body)

	assert.True(t, body.Security.Enabled)
	assert.Equal(t, config.SecurityActionReject, body.Security.Action)
	assert.Equal(t, 5, body.Security.RateLimit.IP.Minute)
	assert.Equal(t, 50, body.Security.RateLimit.IP.Hour)
	assert.Equal(t, 500, body.Security.RateLimit.IP.Day)
	assert.Equal(t, 30, body.Security.RateLimit.Global.Minute)
	assert.True(t, body.Security.AutoBan.Enabled)
	assert.Equal(t, 3, body.Security.AutoBan.Threshold)
	assert.Equal(t, []string{"203.0.113.9", "10.0.0.0/8"}, body.Security.Blacklist)
	assert.Equal(t, []string{"127.0.0.1"}, body.Security.Whitelist)
	assert.Equal(t, []string{"/captcha/verify", "/webhook"}, body.Security.ExcludePaths, "trailing slash is trimmed")
	assert.Equal(t, "geetest", body.Captcha.CaptchaType)
	assert.Equal(t, "gt-id", body.Captcha.Geetest.CaptchaID)
	assert.Equal(t, "gt-key", body.Captcha.Geetest.CaptchaKey)
	assert.Equal(t, "ts-site", body.Captcha.Turnstile.SiteKey)
	assert.Equal(t, "ts-secret", body.Captcha.Turnstile.SecretKey)

	// 落盘的配置文件应保留原有注释与结构
	raw, err := os.ReadFile(confFile)
	require.NoError(t, err)
	content := string(raw)
	assert.Contains(t, content, "# Security (IP ban, rate limit, captcha challenge, attack report)")
	assert.Contains(t, content, "# Upload")
	assert.Contains(t, content, "# Geetest (https://www.geetest.com)")

	reloaded, err := config.NewFromFile(confFile)
	require.NoError(t, err)
	assert.Equal(t, 5, reloaded.Security.RateLimit.IP.Minute)
	assert.Equal(t, config.TypeGeetest, reloaded.Captcha.CaptchaType)
	assert.Equal(t, "gt-key", reloaded.Captcha.Geetest.CaptchaKey)
	assert.Equal(t, "ts-secret", reloaded.Captcha.Turnstile.SecretKey)
	assert.Equal(t, []string{"/captcha/verify", "/webhook"}, reloaded.Security.ExcludePaths)
}

func TestSecurityConfigSaveRejectsInvalidValues(t *testing.T) {
	confFile := copyExampleConfig(t)
	router, _, token := securityTestApp(t, confFile)

	cases := []map[string]any{
		{"action": "nuke"},
		{"blacklist": []string{"not-an-ip"}},
		{"whitelist": []string{"1.2.3.4/99"}},
		{"event_retention_days": -1},
		{"auto_ban": map[string]any{"threshold": -2}},
		{"captcha": map[string]any{"captcha_type": "unknown"}},
		{"exclude_paths": []string{"captcha/verify"}},
	}

	for _, payload := range cases {
		resp, err := securityRequest(router, http.MethodPut, "/security/config", payload, token, "203.0.113.10")
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "payload %v should be rejected", payload)
	}
}

func TestSecurityEventListAndReport(t *testing.T) {
	router, testApp, token := securityTestApp(t, "")

	events := []entity.SecurityEvent{
		{IP: "203.0.113.1", Kind: string(entity.SecurityEventKindRateLimit), Scope: entity.SecurityScopeIP, Window: "minute", Action: "reject", Path: "/api/v2/comments", Method: "POST", Blocked: true},
		{IP: "203.0.113.1", Kind: string(entity.SecurityEventKindRateLimit), Scope: entity.SecurityScopeIP, Window: "hour", Action: "reject", Path: "/api/v2/comments", Method: "POST", Blocked: true},
		{IP: "203.0.113.2", Kind: string(entity.SecurityEventKindCaptchaFailed), Action: "captcha", Path: "/api/v2/comments", Method: "POST", Blocked: true},
	}
	require.NoError(t, testApp.Dao().DB().Create(&events).Error)

	type eventList struct {
		Total  int64                        `json:"count"`
		Events []entity.CookedSecurityEvent `json:"events"`
	}

	resp, err := securityRequest(router, http.MethodGet, "/security/events", map[string]any{}, token, "203.0.113.10")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var list eventList
	readJSON(t, resp, &list)
	assert.Equal(t, int64(3), list.Total)
	require.Len(t, list.Events, 3)
	assert.NotEmpty(t, list.Events[0].Date)

	resp, err = securityRequest(router, http.MethodGet, "/security/events?kind=captcha_failed", map[string]any{}, token, "203.0.113.10")
	require.NoError(t, err)
	readJSON(t, resp, &list)
	assert.Equal(t, int64(1), list.Total)
	require.Len(t, list.Events, 1)
	assert.Equal(t, "203.0.113.2", list.Events[0].IP)

	resp, err = securityRequest(router, http.MethodGet, "/security/events?ip=203.0.113.1", map[string]any{}, token, "203.0.113.10")
	require.NoError(t, err)
	readJSON(t, resp, &list)
	assert.Equal(t, int64(2), list.Total)

	// 报表
	resp, err = securityRequest(router, http.MethodGet, "/security/report?days=7", map[string]any{}, token, "203.0.113.10")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var report struct {
		RangeDays int   `json:"range_days"`
		Total     int64 `json:"total"`
		Blocked   int64 `json:"blocked"`
		ByKind    []struct {
			Kind  string `json:"kind"`
			Count int64  `json:"count"`
		} `json:"by_kind"`
		ByDay []struct {
			Date  string `json:"date"`
			Count int64  `json:"count"`
		} `json:"by_day"`
		TopIPs []struct {
			IP    string `json:"ip"`
			Count int64  `json:"count"`
		} `json:"top_ips"`
		TopPaths []struct {
			Path  string `json:"path"`
			Count int64  `json:"count"`
		} `json:"top_paths"`
	}
	readJSON(t, resp, &report)
	assert.Equal(t, 7, report.RangeDays)
	assert.Equal(t, int64(3), report.Total)
	assert.Equal(t, int64(3), report.Blocked)
	require.NotEmpty(t, report.ByKind)
	assert.Equal(t, "rate_limit", report.ByKind[0].Kind)
	assert.Equal(t, int64(2), report.ByKind[0].Count)
	require.Len(t, report.ByDay, 1)
	// 报表按配置时区的自然日归组，而不是服务器进程的 TZ
	assert.Equal(t, events[0].CreatedAt.In(time.Local).Format("2006-01-02"), report.ByDay[0].Date)
	assert.Equal(t, int64(3), report.ByDay[0].Count)
	require.NotEmpty(t, report.TopIPs)
	assert.Equal(t, "203.0.113.1", report.TopIPs[0].IP)
	assert.Equal(t, int64(2), report.TopIPs[0].Count)
	require.NotEmpty(t, report.TopPaths)
	assert.Equal(t, "/api/v2/comments", report.TopPaths[0].Path)

	// 删除单条
	resp, err = securityRequest(router, http.MethodDelete, "/security/events/"+strconv.FormatUint(uint64(events[0].ID), 10), map[string]any{}, token, "203.0.113.10")
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	resp, err = securityRequest(router, http.MethodGet, "/security/events", map[string]any{}, token, "203.0.113.10")
	require.NoError(t, err)
	readJSON(t, resp, &list)
	assert.Equal(t, int64(2), list.Total)

	// 清空
	resp, err = securityRequest(router, http.MethodDelete, "/security/events", map[string]any{}, token, "203.0.113.10")
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	resp, err = securityRequest(router, http.MethodGet, "/security/events", map[string]any{}, token, "203.0.113.10")
	require.NoError(t, err)
	readJSON(t, resp, &list)
	assert.Equal(t, int64(0), list.Total)
}

func TestSecurityBansLifecycle(t *testing.T) {
	router, _, token := securityTestApp(t, "")

	resp, err := securityRequest(router, http.MethodPost, "/security/bans", map[string]any{
		"ip":       "198.51.100.23",
		"reason":   "spam comments",
		"note":     "from the attack report",
		"duration": 3600,
	}, token, "203.0.113.10")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var created entity.CookedIPBan
	readJSON(t, resp, &created)
	assert.NotZero(t, created.ID)
	assert.Equal(t, "198.51.100.23", created.IP)
	assert.True(t, created.Active)
	assert.NotEmpty(t, created.ExpireAt)
	assert.Equal(t, "from the attack report", created.Note)

	type banList struct {
		Total int64                `json:"count"`
		Bans  []entity.CookedIPBan `json:"bans"`
	}

	resp, err = securityRequest(router, http.MethodGet, "/security/bans", map[string]any{}, token, "203.0.113.10")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var list banList
	readJSON(t, resp, &list)
	assert.Equal(t, int64(1), list.Total)
	require.Len(t, list.Bans, 1)
	assert.Equal(t, "spam comments", list.Bans[0].Reason)

	// 非法目标无法创建
	resp, err = securityRequest(router, http.MethodPost, "/security/bans", map[string]any{"ip": "nope"}, token, "203.0.113.10")
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// 解除封禁
	resp, err = securityRequest(router, http.MethodDelete, "/security/bans/"+strconv.FormatUint(uint64(created.ID), 10), map[string]any{}, token, "203.0.113.10")
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	resp, err = securityRequest(router, http.MethodGet, "/security/bans", map[string]any{}, token, "203.0.113.10")
	require.NoError(t, err)
	readJSON(t, resp, &list)
	assert.Equal(t, int64(0), list.Total)
	assert.Empty(t, list.Bans)
}
