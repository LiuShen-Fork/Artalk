package handler_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/artalkjs/artalk/v2/internal/config"
	"github.com/artalkjs/artalk/v2/internal/core"
	"github.com/artalkjs/artalk/v2/internal/entity"
	"github.com/artalkjs/artalk/v2/server/handler"
	"github.com/artalkjs/artalk/v2/server/middleware/limiter"
	"github.com/stretchr/testify/require"
)

// A blocked sender must not learn which keyword matched, otherwise the next
// attempt just obfuscates that word. Admins keep the detail in the log.
func TestKeywordInterceptResponseHidesKeyword(t *testing.T) {
	app, router := NewApiTestApp()
	t.Cleanup(func() { require.NoError(t, app.Cleanup()) })
	app.Conf().Captcha.Enabled = false
	router.Use(limiter.ActionLimitMiddleware(app.App, limiter.ActionLimitConf{}))
	handler.CommentCreate(app.App, router)

	const keyword = "hidden-block-word"
	app.Conf().Moderator.Intercept = config.CommentInterceptConf{Enabled: true, Keywords: keyword}
	// The service copies the moderator config on construction, so rebuild it
	// with the enabled interceptor instead of editing the conf behind its back.
	svc := core.NewAntiSpamService(app.App)
	require.NoError(t, svc.Init())
	core.AppInject(app.App, svc)

	payload, err := json.Marshal(handler.ParamsCommentCreate{
		Name: "Visitor", Email: "visitor@example.org", Content: "please read " + keyword,
		SiteName: "Site A", PageKey: "/keyword-intercept",
	})
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/comments", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	resp, err := router.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	body := string(raw)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	require.Contains(t, body, "security rules")
	require.NotContains(t, body, keyword, "the response must not name the matched keyword")

	// Nothing was persisted.
	var count int64
	require.NoError(t, app.Dao().DB().Model(&entity.Comment{}).
		Where("page_key = ?", "/keyword-intercept").Count(&count).Error)
	require.Zero(t, count)

	// The moderation log keeps the detail for the administrator.
	var logs []entity.ModerationLog
	require.NoError(t, app.Dao().DB().Where("checker = ?", "intercept").Find(&logs).Error)
	require.Len(t, logs, 1)
	require.Contains(t, logs[0].Message, keyword)
}
