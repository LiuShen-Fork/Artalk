package common

import (
	"net"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
)

func TestClientIP(t *testing.T) {
	app := fiber.New(fiber.Config{ProxyHeader: "X-Forwarded-For"})
	defer func() { _ = app.Shutdown() }()

	newCtx := func(t *testing.T) *fiber.Ctx {
		t.Helper()
		return app.AcquireCtx(&fasthttp.RequestCtx{})
	}

	t.Run("proxy header wins", func(t *testing.T) {
		c := newCtx(t)
		defer app.ReleaseCtx(c)
		c.Request().Header.Set("X-Forwarded-For", "203.0.113.9")

		assert.Equal(t, "203.0.113.9", ClientIP(c))
	})

	t.Run("falls back to the connection address", func(t *testing.T) {
		c := newCtx(t)
		defer app.ReleaseCtx(c)
		c.Context().SetRemoteAddr(&net.TCPAddr{IP: net.ParseIP("198.51.100.7"), Port: 45678})

		// Fiber reports an empty IP when the trusted header is missing; the
		// connection address is still a real client address.
		assert.Equal(t, "198.51.100.7", ClientIP(c))
	})

	t.Run("unknown connection address", func(t *testing.T) {
		c := newCtx(t)
		defer app.ReleaseCtx(c)
		c.Context().SetRemoteAddr(&net.TCPAddr{IP: net.IPv4zero, Port: 45678})

		assert.Equal(t, "", ClientIP(c))
	})
}
