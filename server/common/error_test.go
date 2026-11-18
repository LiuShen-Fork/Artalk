package common_test

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/artalkjs/artalk/v2/server/common"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrorHandlerKeepsInternalDetailsOutOfTheResponse(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: common.ErrorHandler})
	app.Get("/boom", func(c *fiber.Ctx) error {
		return errors.New(`sqlite3: no such table: "users"`)
	})
	app.Get("/bad", func(c *fiber.Ctx) error {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "unprocessable entity")
	})

	get := func(path string) (int, string) {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		return resp.StatusCode, string(body)
	}

	code, body := get("/boom")
	assert.Equal(t, http.StatusInternalServerError, code)
	assert.NotContains(t, body, "sqlite3", "a 5xx must not describe the server's internals to the caller")
	assert.Contains(t, body, "internal server error")

	// 4xx 的文案是写给调用方看的，原样返回
	code, body = get("/bad")
	assert.Equal(t, http.StatusUnprocessableEntity, code)
	assert.True(t, strings.Contains(body, "unprocessable entity"), body)
}
