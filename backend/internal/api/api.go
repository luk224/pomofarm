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

	// The Flow table never changes while the server runs: clients may cache it.
	app.Get("/api/flow", func(c *fiber.Ctx) error {
		c.Set("Cache-Control", "public, max-age=3600")
		return c.JSON(game.FlowTable())
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
		if err := c.BodyParser(&req); err != nil || (req.Kind != "seed" && req.Kind != "animal") {
			return fail(c, service.ErrInvalid)
		}
		var err error
		if req.Kind == "animal" {
			err = svc.UnlockAnimal(c.UserContext(), req.Key)
		} else {
			err = svc.UnlockSeed(c.UserContext(), req.Key)
		}
		if err != nil {
			return fail(c, err)
		}
		return respondState(c, svc, fiber.StatusCreated)
	})

	// Structures bought with 🪙: hives (placed beside a plot) and the Dog.
	app.Post("/api/structures", func(c *fiber.Ctx) error {
		var req struct {
			Kind   string `json:"kind"`
			PlotID int64  `json:"plot_id"`
		}
		if err := c.BodyParser(&req); err != nil {
			return fail(c, service.ErrInvalid)
		}
		var err error
		switch req.Kind {
		case "hive":
			err = svc.BuyHive(c.UserContext(), req.PlotID)
		case "dog":
			err = svc.BuyDog(c.UserContext())
		default:
			err = service.ErrInvalid
		}
		if err != nil {
			return fail(c, err)
		}
		return respondState(c, svc, fiber.StatusCreated)
	})

	app.Post("/api/structures/:id/move", func(c *fiber.Ctx) error {
		id, err := c.ParamsInt("id")
		if err != nil {
			return fail(c, service.ErrInvalid)
		}
		var req struct {
			PlotID int64 `json:"plot_id"`
		}
		if err := c.BodyParser(&req); err != nil {
			return fail(c, service.ErrInvalid)
		}
		if err := svc.MoveHive(c.UserContext(), int64(id), req.PlotID); err != nil {
			return fail(c, err)
		}
		return respondState(c, svc, fiber.StatusOK)
	})

	app.Post("/api/plots/:id/clear", func(c *fiber.Ctx) error {
		id, err := c.ParamsInt("id")
		if err != nil {
			return fail(c, service.ErrInvalid)
		}
		var req struct {
			Confirm bool `json:"confirm"`
		}
		_ = c.BodyParser(&req) // body is optional
		if err := svc.ClearPlot(c.UserContext(), int64(id), req.Confirm); err != nil {
			return fail(c, err)
		}
		return respondState(c, svc, fiber.StatusOK)
	})

	app.Post("/api/plots", func(c *fiber.Ctx) error {
		if err := svc.BuyPlot(c.UserContext()); err != nil {
			return fail(c, err)
		}
		return respondState(c, svc, fiber.StatusCreated)
	})

	app.Post("/api/silo/upgrade", func(c *fiber.Ctx) error {
		if err := svc.UpgradeSilo(c.UserContext()); err != nil {
			return fail(c, err)
		}
		return respondState(c, svc, fiber.StatusOK)
	})

	app.Post("/api/silo/collect", func(c *fiber.Ctx) error {
		milli, err := svc.CollectSilo(c.UserContext())
		if err != nil {
			return fail(c, err)
		}
		st, err := svc.State(c.UserContext())
		if err != nil {
			return fail(c, err)
		}
		return c.JSON(fiber.Map{"collected_milli": milli, "state": st})
	})

	app.Post("/api/rest/skip", func(c *fiber.Ctx) error {
		if err := svc.SkipRest(c.UserContext()); err != nil {
			return fail(c, err)
		}
		return respondState(c, svc, fiber.StatusOK)
	})

	app.Post("/api/settings", func(c *fiber.Ctx) error {
		var req struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		}
		if err := c.BodyParser(&req); err != nil {
			return fail(c, service.ErrInvalid)
		}
		if err := svc.SetSetting(c.UserContext(), req.Key, req.Value); err != nil {
			return fail(c, err)
		}
		return respondState(c, svc, fiber.StatusOK)
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
	service.ErrHarvestFirst:      fiber.StatusConflict,
	service.ErrSiloEmpty:         fiber.StatusConflict,
	service.ErrMaxedOut:          fiber.StatusConflict,
	service.ErrNoRest:            fiber.StatusConflict,
	service.ErrInsufficientCoins: fiber.StatusConflict,
	service.ErrAlreadyOwned:      fiber.StatusConflict,
	service.ErrCellTaken:         fiber.StatusConflict,
	service.ErrAnimalLocked:      fiber.StatusForbidden,
	service.ErrNeedsConfirmation: fiber.StatusConflict,
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
