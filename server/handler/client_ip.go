package handler

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

// clientIP reads the requester IP.
//
// Artalk trusts the configured proxy header by default (`X-Forwarded-For`), and
// Fiber then reports an empty IP when that header is missing — a direct
// connection, or a reverse proxy that does not forward it. Falling back to the
// connection address keeps the recorded IP (and the per-IP limits) working in
// both setups instead of silently storing an empty string.
func clientIP(c *fiber.Ctx) string {
	if ip := strings.TrimSpace(c.IP()); ip != "" {
		return ip
	}
	if addr := c.Context().RemoteIP(); addr != nil && !addr.IsUnspecified() {
		return addr.String()
	}
	return ""
}
