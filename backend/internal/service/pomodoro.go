package service

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/luk224/pomofarm/backend/internal/game"
)

// PlantRequest starts a Pomodoro on a plot. DurationMin is only honoured for
// the Oak (Flow mode, 60–120 min); other crops have a fixed duration.
type PlantRequest struct {
	PlotID      int64  `json:"plot_id"`
	PlantType   string `json:"plant_type"`
	Tag         string `json:"tag"`
	DurationMin int    `json:"duration_min"`
}

// Plant starts a Pomodoro. Only one may be active; a second attempt (e.g. from
// another device) gets ErrPomodoroActive.
func (s *Service) Plant(ctx context.Context, req PlantRequest) error {
	crop, ok := game.CropByKey(req.PlantType)
	if !ok {
		return ErrInvalid
	}
	durMin, lifeS := crop.DurationMin, int64(crop.LifeH)*3600
	if crop.Key == "oak" && req.DurationMin != 0 {
		if req.DurationMin < game.FlowMinMin || req.DurationMin > game.FlowMaxMin {
			return ErrInvalid
		}
		durMin, lifeS = req.DurationMin, game.FlowLifeSeconds(req.DurationMin)
	} else if req.DurationMin != 0 && req.DurationMin != crop.DurationMin {
		return ErrInvalid
	}
	tag := strings.TrimSpace(req.Tag)
	if len(tag) > 60 {
		return ErrInvalid
	}

	return s.withTx(ctx, func(tx *sql.Tx, now time.Time) error {
		if err := s.settle(ctx, tx, now); err != nil {
			return err
		}
		if a, err := loadActive(ctx, tx); err != nil {
			return err
		} else if a != nil {
			return ErrPomodoroActive
		}
		if crop.Unlock > 0 {
			var n int
			if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM unlocks WHERE player_id = ? AND kind = 'seed' AND key = ?`,
				PlayerID, crop.Key).Scan(&n); err != nil {
				return err
			}
			if n == 0 {
				return ErrSeedLocked
			}
		}
		var state string
		var version, harvested int64
		err := tx.QueryRowContext(ctx, `SELECT state, harvested, version FROM plots WHERE id = ? AND player_id = ?`, req.PlotID, PlayerID).
			Scan(&state, &harvested, &version)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if state != "empty" && state != "withered" {
			return ErrPlotBusy
		}
		if state == "withered" && harvested == 0 {
			return ErrHarvestFirst // never plant over a plant whose 💧 reward is still waiting
		}
		var tagID any
		if tag != "" {
			if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO tags (player_id, name) VALUES (?, ?)`, PlayerID, tag); err != nil {
				return err
			}
			var id int64
			if err := tx.QueryRowContext(ctx, `SELECT id FROM tags WHERE player_id = ? AND name = ?`, PlayerID, tag).Scan(&id); err != nil {
				return err
			}
			tagID = id
		}
		plannedS := int64(durMin) * 60
		res, err := tx.ExecContext(ctx, `INSERT INTO pomodoros (player_id, plot_id, plant_type, tag_id, planned_s, started_at, status)
			VALUES (?, ?, ?, ?, ?, ?, 'running')`, PlayerID, req.PlotID, crop.Key, tagID, plannedS, fmtTime(now))
		if isUnique(err) {
			return ErrPomodoroActive
		}
		if err != nil {
			return err
		}
		pid, _ := res.LastInsertId()
		upd, err := tx.ExecContext(ctx, `UPDATE plots SET state = 'growing', plant_type = ?, planted_at = ?, grow_s = ?,
			matured_at = NULL, harvested = 0, life_s = ?, wilts_at = NULL, collected_to = NULL, version = version + 1
			WHERE id = ? AND version = ?`, crop.Key, fmtTime(now), plannedS, lifeS, req.PlotID, version)
		if err != nil {
			return err
		}
		if n, _ := upd.RowsAffected(); n != 1 {
			return ErrConflict
		}
		if err := endRest(ctx, tx); err != nil { // starting a new Pomodoro ends any break
			return err
		}
		return addEvent(ctx, tx, pid, "start", now)
	})
}

// mutateActive loads the active Pomodoro (after settling), applies op, saves it and logs the event.
func (s *Service) mutateActive(ctx context.Context, kind string, op func(*activeRow, time.Time) error) error {
	return s.withTx(ctx, func(tx *sql.Tx, now time.Time) error {
		if err := s.settle(ctx, tx, now); err != nil {
			return err
		}
		a, err := loadActive(ctx, tx)
		if err != nil {
			return err
		}
		if a == nil {
			return ErrNoActivePomodoro
		}
		if err := op(a, now); err != nil {
			if errors.Is(err, game.ErrNotRunning) || errors.Is(err, game.ErrNotPaused) {
				return ErrWrongState
			}
			return err
		}
		if err := saveActive(ctx, tx, a); err != nil {
			return err
		}
		if kind == "cancel" && a.PlotID.Valid {
			if _, err := tx.ExecContext(ctx, `UPDATE plots SET state = 'empty', plant_type = NULL, planted_at = NULL, grow_s = NULL,
				matured_at = NULL, harvested = 0, life_s = NULL, wilts_at = NULL, collected_to = NULL, version = version + 1
				WHERE id = ?`, a.PlotID.Int64); err != nil {
				return err
			}
		}
		return addEvent(ctx, tx, a.ID, kind, now)
	})
}

func (s *Service) Pause(ctx context.Context) error {
	return s.mutateActive(ctx, "pause", func(a *activeRow, now time.Time) error { return a.P.Pause(now) })
}

func (s *Service) Resume(ctx context.Context) error {
	return s.mutateActive(ctx, "resume", func(a *activeRow, now time.Time) error { return a.P.Resume(now) })
}

// Cancel abandons the Pomodoro: the plant disappears and no 💧 are given (GDD §3.3).
func (s *Service) Cancel(ctx context.Context) error {
	return s.mutateActive(ctx, "cancel", func(a *activeRow, now time.Time) error { return a.P.Cancel(now) })
}

// Harvest collects the 💧 reward of a mature plant, once. The plant stays (GDD §3.2).
func (s *Service) Harvest(ctx context.Context, plotID int64) (reward int, err error) {
	err = s.withTx(ctx, func(tx *sql.Tx, now time.Time) error {
		if err := s.settle(ctx, tx, now); err != nil {
			return err
		}
		var state, plant string
		var harvested, version, growS int64
		var maturedAt sql.NullString
		err := tx.QueryRowContext(ctx, `SELECT state, COALESCE(plant_type,''), harvested, version, COALESCE(grow_s, 0), matured_at FROM plots WHERE id = ? AND player_id = ?`,
			plotID, PlayerID).Scan(&state, &plant, &harvested, &version, &growS, &maturedAt)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if harvested == 1 {
			return ErrAlreadyHarvested
		}
		if state != "mature" && state != "withered" { // a withered plant can still be harvested
			return ErrNotMature
		}
		// The reward follows from what was planted, which the plot itself remembers (type and duration).
		crop, ok := game.CropByKey(plant)
		if !ok {
			return ErrInvalid
		}
		reward = crop.Reward
		if plant == "oak" && int(growS/60) != crop.DurationMin && growS > 0 {
			reward = game.FlowReward(int(growS / 60))
		}
		upd, err := tx.ExecContext(ctx, `UPDATE plots SET harvested = 1, version = version + 1 WHERE id = ? AND version = ?`, plotID, version)
		if err != nil {
			return err
		}
		if n, _ := upd.RowsAffected(); n != 1 {
			return ErrConflict
		}
		if _, err := tx.ExecContext(ctx, `UPDATE pomodoros SET reward_focus = ? WHERE id = (SELECT id FROM pomodoros WHERE plot_id = ? AND status = 'completed' ORDER BY id DESC LIMIT 1)`, reward, plotID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE players SET focus_points = focus_points + ?, lifetime_focus = lifetime_focus + ?,
			version = version + 1 WHERE id = ?`, reward, reward, PlayerID); err != nil {
			return err
		}
		var matured time.Time
		if maturedAt.Valid {
			matured = parseTime(maturedAt.String)
		}
		return startRest(ctx, tx, now, growS, matured)
	})
	return reward, err
}
