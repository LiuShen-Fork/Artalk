package auth

import (
	"embed"
	"html/template"

	"github.com/gofiber/fiber/v2"
)

//go:embed callback.html bind_callback.html
var callbackPageHTML embed.FS

func ResponseCallbackPage(c *fiber.Ctx, token string) error {
	h, err := callbackPageHTML.ReadFile("callback.html")
	if err != nil {
		return err
	}
	t, err := template.New("callback").Parse(string(h))
	if err != nil {
		return err
	}

	c.Set(fiber.HeaderCacheControl, "no-cache, no-store, must-revalidate")
	c.Set(fiber.HeaderPragma, "no-cache")
	c.Set(fiber.HeaderExpires, "0")
	c.Set(fiber.HeaderContentType, "text/html; charset=utf-8")
	return t.Execute(c.Response().BodyWriter(), fiber.Map{
		"token": token,
	})
}

// ResponseBindCallbackPage closes the OAuth popup opened by the user console
// and tells the opener whether the identity was linked.
func ResponseBindCallbackPage(c *fiber.Ctx, provider string, ok bool, message string) error {
	h, err := callbackPageHTML.ReadFile("bind_callback.html")
	if err != nil {
		return err
	}
	t, err := template.New("bind_callback").Parse(string(h))
	if err != nil {
		return err
	}

	c.Set(fiber.HeaderCacheControl, "no-cache, no-store, must-revalidate")
	c.Set(fiber.HeaderPragma, "no-cache")
	c.Set(fiber.HeaderExpires, "0")
	c.Set(fiber.HeaderContentType, "text/html; charset=utf-8")
	return t.Execute(c.Response().BodyWriter(), fiber.Map{
		"ok":       ok,
		"provider": provider,
		"message":  message,
	})
}
