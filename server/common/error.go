package common

import (
	"errors"
	"fmt"

	"github.com/artalkjs/artalk/v2/internal/log"
	"github.com/gofiber/fiber/v2"
)

func ErrorHandler(ctx *fiber.Ctx, err error) error {
	// Status code defaults to 500
	code := fiber.StatusInternalServerError

	// Retrieve the custom status code if it's a *fiber.Error
	var e *fiber.Error
	if errors.As(err, &e) {
		code = e.Code
	}

	// A 4xx message was written for the caller (bad params, unsupported
	// content type, ...), so it is passed through. A 5xx one is a server-side
	// failure whose text names tables, paths or hosts — useless to the caller
	// and useful to whoever is probing: it goes to the log instead.
	if code >= fiber.StatusInternalServerError {
		log.Errorf("Request failed: %s %s: %s", ctx.Method(), ctx.Path(), err.Error())
		return RespError(ctx, code, "Error: internal server error")
	}

	return RespError(ctx, code, fmt.Sprintf("Error: %s", err.Error()))
}
