package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/artalkjs/artalk/v2/internal/entity"
	"github.com/artalkjs/artalk/v2/server/common"
	"github.com/artalkjs/artalk/v2/server/handler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModerationLogManagement(t *testing.T) {
	app, fiberApp := NewApiTestApp()
	defer app.Cleanup()

	admin := app.Dao().FindUserByID(1000)
	token, err := common.LoginGetUserToken(admin, app.Conf().AppKey, 3600)
	require.NoError(t, err)

	logs := []entity.ModerationLog{
		{CommentID: 1000, SiteName: "Site A", Checker: "ai", Status: "pass", Action: "allow"},
		{CommentID: 1000, SiteName: "Site A", Checker: "keywords", Status: "pass", Action: "replace"},
		{CommentID: 1000, SiteName: "Site A", Checker: "ai", Status: "block", Action: "pending"},
		{CommentID: 1000, SiteName: "Site B", Checker: "ai", Status: "error", Action: "pending"},
	}
	for i := range logs {
		require.NoError(t, app.Dao().DB().Create(&logs[i]).Error)
	}

	handler.ModerationLogList(app.App, fiberApp)
	handler.ModerationLogDelete(app.App, fiberApp)
	handler.ModerationLogClear(app.App, fiberApp)

	listReq := httptest.NewRequest(http.MethodGet, "/moderation/logs?site_name=Site%20A&token="+token, nil)
	listResp, err := fiberApp.Test(listReq)
	require.NoError(t, err)
	defer listResp.Body.Close()
	require.Equal(t, http.StatusOK, listResp.StatusCode)

	var list struct {
		Count int64 `json:"count"`
		Logs  []struct {
			ID     uint   `json:"id"`
			Status string `json:"status"`
			Action string `json:"action"`
		} `json:"logs"`
	}
	require.NoError(t, json.NewDecoder(listResp.Body).Decode(&list))
	assert.Equal(t, int64(2), list.Count)
	assert.Len(t, list.Logs, 2)
	assert.NotContains(t, []string{list.Logs[0].Action, list.Logs[1].Action}, "allow")

	deleteReq := httptest.NewRequest(http.MethodDelete, "/moderation/logs/"+jsonNumber(logs[1].ID)+"?token="+token, nil)
	deleteResp, err := fiberApp.Test(deleteReq)
	require.NoError(t, err)
	defer deleteResp.Body.Close()
	assert.Equal(t, http.StatusOK, deleteResp.StatusCode)
	var deleted entity.ModerationLog
	assert.Error(t, app.Dao().DB().First(&deleted, logs[1].ID).Error)

	clearReq := httptest.NewRequest(http.MethodDelete, "/moderation/logs?site_name=Site%20A&token="+token, nil)
	clearResp, err := fiberApp.Test(clearReq)
	require.NoError(t, err)
	defer clearResp.Body.Close()
	assert.Equal(t, http.StatusOK, clearResp.StatusCode)

	var siteACount, siteBCount int64
	app.Dao().DB().Model(&entity.ModerationLog{}).Where("site_name = ?", "Site A").Count(&siteACount)
	app.Dao().DB().Model(&entity.ModerationLog{}).Where("site_name = ?", "Site B").Count(&siteBCount)
	assert.Zero(t, siteACount)
	assert.Equal(t, int64(1), siteBCount)

	clearAllReq := httptest.NewRequest(http.MethodDelete, "/moderation/logs?token="+token, nil)
	clearAllResp, err := fiberApp.Test(clearAllReq)
	require.NoError(t, err)
	defer clearAllResp.Body.Close()
	assert.Equal(t, http.StatusOK, clearAllResp.StatusCode)

	var remainingCount int64
	app.Dao().DB().Model(&entity.ModerationLog{}).Count(&remainingCount)
	assert.Zero(t, remainingCount)
}

func jsonNumber(value uint) string {
	return strconv.FormatUint(uint64(value), 10)
}
