package handler_test

import (
	"fmt"

	"github.com/artalkjs/artalk/v2/test"
	"github.com/gofiber/fiber/v2"
)

func NewApiTestApp() (*test.TestApp, *fiber.App) {
	app, err := test.NewTestApp()
	if err != nil {
		// Swallowing this error used to leave nil behind and turn a setup failure
		// into a confusing nil-pointer panic somewhere below.
		panic(fmt.Errorf("failed to create test app: %w", err))
	}
	fiberApp := fiber.New(fiber.Config{
		ProxyHeader: "X-Forwarded-For",
	})
	return app, fiberApp
}
