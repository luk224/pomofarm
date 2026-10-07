// Package api exposes the HTTP layer; rules live in internal/game and internal/service.
package api

import (
	"database/sql"
	"errors"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/luk224/pomofarm/backend/internal/game"
	"github.com/luk224/pomofarm/backend/internal/service"
	"github.com/luk224/pomofarm/backend/internal/store"
)

// New builds the Fiber app with all routes registered.
func New(db *sql.DB, clock game.Clock) *fiber.App {
	app := fiber.New(fiber.Config{DisableStartupMessage: true, BodyLimit: 16 * 1024})
	svc := service.New(db, clock)

	app.Get("/api/health", func(c *fiber.Ctx) error {
		v, err := store.SchemaVersion(db)
		if err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"status": "db_error"})
		}
		return c.JSON(fiber.Map{"status": "ok", "schema_version": v})
	})

	app.Get("/api/state", func(c *fiber.Ctx) error {
		st, err := svc.State(c.UserContext())
		if err != nil {
			return fail(c, err)
		}
		return c.JSON(st)
	})

	app.Post("/api/pomodoros", func(c *fiber.Ctx) error {
		var req service.PlantRequest
		if err := c.BodyParser(&req); err != nil {
			return fail(c, service.ErrInvalid)
		}
		if err := svc.Plant(c.UserContext(), req); err != nil {
			return fail(c, err)
		}
		return respondState(c, svc, fiber.StatusCreated)
	})

	for name, op := range map[string]func(*service.Service, *fiber.Ctx) error{
		"pause":  func(s *service.Service, c *fiber.Ctx) error { return s.Pause(c.UserContext()) },
		"resume": func(s *service.Service, c *fiber.Ctx) error { return s.Resume(c.UserContext()) },
		"cancel": func(s *service.Service, c *fiber.Ctx) error { return s.Cancel(c.UserContext()) },
	} {
		op := op
		app.Post("/api/pomodoros/active/"+name, func(c *fiber.Ctx) error {
			if err := op(svc, c); err != nil {
				return fail(c, err)
			}
			return respondState(c, svc, fiber.StatusOK)
		})
	}

	app.Post("/api/unlocks", func(c *fiber.Ctx) error {
		var req struct {
			Kind string `json:"kind"`
			Key  string `json:"key"`
		}
		if err := c.BodyParser(&req); err != nil || req.Kind != "seed" {
			return fail(c, service.ErrInvalid)
		}
		if err := svc.UnlockSeed(c.UserContext(), req.Key); err != nil {
			return fail(c, err)
		}
		return respondState(c, svc, fiber.StatusCreated)
	})

	app.Post("/api/plots/:id/harvest", func(c *fiber.Ctx) error {
		id, err := c.ParamsInt("id")
		if err != nil {
			return fail(c, service.ErrInvalid)
		}
		reward, err := svc.Harvest(c.UserContext(), int64(id))
		if err != nil {
			return fail(c, err)
		}
		st, err := svc.State(c.UserContext())
		if err != nil {
			return fail(c, err)
		}
		return c.JSON(fiber.Map{"reward_focus": reward, "state": st})
	})
	return app
}

func respondState(c *fiber.Ctx, svc *service.Service, status int) error {
	st, err := svc.State(c.UserContext())
	if err != nil {
		return fail(c, err)
	}
	return c.Status(status).JSON(st)
}

var statusOf = map[error]int{
	service.ErrNoPlayer:          fiber.StatusNotFound,
	service.ErrNotFound:          fiber.StatusNotFound,
	service.ErrInvalid:           fiber.StatusBadRequest,
	service.ErrSeedLocked:        fiber.StatusForbidden,
	service.ErrPomodoroActive:    fiber.StatusConflict,
	service.ErrNoActivePomodoro:  fiber.StatusConflict,
	service.ErrPlotBusy:          fiber.StatusConflict,
	service.ErrNotMature:         fiber.StatusConflict,
	service.ErrAlreadyHarvested:  fiber.StatusConflict,
	service.ErrWrongState:        fiber.StatusConflict,
	service.ErrConflict:          fiber.StatusConflict,
	service.ErrAlreadyUnlocked:   fiber.StatusConflict,
	service.ErrInsufficientFocus: fiber.StatusConflict,
}

// fail writes {"error": code}. Unknown errors are logged and hidden behind a 500.
func fail(c *fiber.Ctx, err error) error {
	for target, status := range statusOf {
		if errors.Is(err, target) {
			return c.Status(status).JSON(fiber.Map{"error": target.Error()})
		}
	}
	log.Printf("%s %s: %v", c.Method(), c.Path(), err)
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal"})
}
