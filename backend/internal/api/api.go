// Package api exposes the HTTP layer; domain logic lives in internal/game.
package api

import (
	"database/sql"

	"github.com/gofiber/fiber/v2"
	"github.com/luk224/pomofarm/backend/internal/store"
)

// New builds the Fiber app with all routes registered.
func New(db *sql.DB) *fiber.App {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Get("/api/health", func(c *fiber.Ctx) error {
		v, err := store.SchemaVersion(db)
		if err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"status": "db_error"})
		}
		return c.JSON(fiber.Map{"status": "ok", "schema_version": v})
	})
	return app
}
