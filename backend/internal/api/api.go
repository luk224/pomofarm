// Package api exposes the HTTP layer; domain logic lives in internal/game.
package api

import "github.com/gofiber/fiber/v2"

// New builds the Fiber app with all routes registered.
func New() *fiber.App {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Get("/api/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})
	return app
}
