package handler_test

import (
	"net/http"
	"regexp"
	"testing"

	"github.com/artalkjs/artalk/v2/server/common"
	"github.com/artalkjs/artalk/v2/server/handler"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type userActivityBody struct {
	Count     int64  `json:"count"`
	Pending   int64  `json:"pending"`
	Votes     int64  `json:"votes"`
	FirstDate string `json:"first_date"`
	ByDay     []struct {
		Date  string `json:"date"`
		Count int64  `json:"count"`
	} `json:"by_day"`
	Recent []struct {
		ID      uint   `json:"id"`
		UserID  uint   `json:"user_id"`
		PageKey string `json:"page_key"`
	} `json:"recent"`
}

var isoDay = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

func userActivityTestApp(t *testing.T) (*fiber.App, string, string) {
	t.Helper()

	app, router := NewApiTestApp()
	t.Cleanup(func() { _ = app.Cleanup() })
	handler.UserActivity(app.App, router)

	tokenA, err := common.LoginGetUserToken(app.Dao().FindUserByID(1000), app.Conf().AppKey, 60)
	require.NoError(t, err)
	tokenB, err := common.LoginGetUserToken(app.Dao().FindUserByID(1001), app.Conf().AppKey, 60)
	require.NoError(t, err)

	return router, tokenA, tokenB
}

func TestUserActivityCountsOnlyOwnComments(t *testing.T) {
	router, tokenA, tokenB := userActivityTestApp(t)

	var a userActivityBody
	resp, err := securityRequest(router, http.MethodGet, "/user/activity?days=30&limit=5", nil, tokenA, "203.0.113.10")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	readJSON(t, resp, &a)

	assert.Greater(t, a.Count, int64(0), "user A has comments in the fixtures")
	assert.LessOrEqual(t, len(a.Recent), 5, "limit caps the recent list")
	assert.Regexp(t, isoDay, a.FirstDate)
	for _, day := range a.ByDay {
		assert.Regexp(t, isoDay, day.Date, "dates are normalised to YYYY-MM-DD")
		assert.Greater(t, day.Count, int64(0))
	}
	for _, item := range a.Recent {
		assert.Equal(t, uint(1000), item.UserID, "the activity feed never leaks other people's comments")
		assert.NotEmpty(t, item.PageKey)
	}

	var b userActivityBody
	resp, err = securityRequest(router, http.MethodGet, "/user/activity", nil, tokenB, "203.0.113.10")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	readJSON(t, resp, &b)

	assert.NotEqual(t, a.Count, b.Count, "each account sees its own totals")
	for _, item := range b.Recent {
		assert.Equal(t, uint(1001), item.UserID)
	}
}

func TestUserActivityWithoutIdentityIsEmpty(t *testing.T) {
	router, _, _ := userActivityTestApp(t)

	resp, err := securityRequest(router, http.MethodGet, "/user/activity", nil, "", "203.0.113.10")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var body userActivityBody
	readJSON(t, resp, &body)
	assert.Zero(t, body.Count)
	assert.Empty(t, body.Recent)
	assert.NotNil(t, body.ByDay, "an empty list, not null, keeps the client simple")
}

func TestUserActivityClampsRange(t *testing.T) {
	router, tokenA, _ := userActivityTestApp(t)

	// Negative and oversized values fall back to the defaults instead of erroring.
	resp, err := securityRequest(router, http.MethodGet, "/user/activity?days=-3&limit=0", nil, tokenA, "203.0.113.10")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var body userActivityBody
	readJSON(t, resp, &body)
	assert.LessOrEqual(t, len(body.Recent), 10)
}
