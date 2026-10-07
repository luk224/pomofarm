package service

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/luk224/pomofarm/backend/internal/game"
)

type PlayerState struct {
	Name          string `json:"name"`
	FocusPoints   int64  `json:"focus_points"`
	LifetimeFocus int64  `json:"lifetime_focus"`
	CoinsMilli    int64  `json:"coins_milli"`
	SiloLevel     int    `json:"silo_level"`
	Season        int    `json:"season"`
}

type PlotState struct {
	ID        int64   `json:"id"`
	X         int     `json:"x"`
	Y         int     `json:"y"`
	State     string  `json:"state"`
	PlantType *string `json:"plant_type"`
	MaturedAt *string `json:"matured_at"`
	WiltsAt   *string `json:"wilts_at"`
	Harvested bool    `json:"harvested"`
}

type PomodoroState struct {
	ID          int64   `json:"id"`
	PlotID      *int64  `json:"plot_id"`
	PlantType   string  `json:"plant_type"`
	Status      string  `json:"status"`
	PlannedS    int64   `json:"planned_s"`
	RemainingMs int64   `json:"remaining_ms"`
	StartedAt   string  `json:"started_at"`
	PausedAt    *string `json:"paused_at"`
}

// SeedState lets the client list seeds and their prices without knowing the balance.
type SeedState struct {
	Key         string `json:"key"`
	DurationMin int    `json:"duration_min"`
	UnlockCost  int    `json:"unlock_cost"`
	Reward      int    `json:"reward"`
	LifeH       int    `json:"life_h"`
	Unlocked    bool   `json:"unlocked"`
}

// State is everything the client needs to draw the game. The client displays it
// and never computes economy or time itself.
type State struct {
	ServerTime string         `json:"server_time"`
	Player     PlayerState    `json:"player"`
	Plots      []PlotState    `json:"plots"`
	Seeds      []SeedState    `json:"seeds"`
	Pomodoro   *PomodoroState `json:"pomodoro"`
}

func (s *Service) State(ctx context.Context) (State, error) {
	var st State
	err := s.withTx(ctx, func(tx *sql.Tx, now time.Time) error {
		if err := s.settle(ctx, tx, now); err != nil {
			return err
		}
		st.ServerTime = fmtTime(now)
		err := tx.QueryRowContext(ctx, `SELECT name, focus_points, lifetime_focus, coins_milli, silo_level, season FROM players WHERE id = ?`, PlayerID).
			Scan(&st.Player.Name, &st.Player.FocusPoints, &st.Player.LifetimeFocus, &st.Player.CoinsMilli, &st.Player.SiloLevel, &st.Player.Season)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNoPlayer
		}
		if err != nil {
			return err
		}
		rows, err := tx.QueryContext(ctx, `SELECT id, x, y, state, plant_type, matured_at, wilts_at, harvested FROM plots WHERE player_id = ? ORDER BY y, x`, PlayerID)
		if err != nil {
			return err
		}
		defer rows.Close()
		st.Plots = []PlotState{}
		for rows.Next() {
			var p PlotState
			var plant, mat, wilt sql.NullString
			var h int
			if err := rows.Scan(&p.ID, &p.X, &p.Y, &p.State, &plant, &mat, &wilt, &h); err != nil {
				return err
			}
			p.PlantType, p.MaturedAt, p.WiltsAt, p.Harvested = strPtr(plant), strPtr(mat), strPtr(wilt), h == 1
			st.Plots = append(st.Plots, p)
		}
		if err := rows.Err(); err != nil {
			return err
		}
		unlocked := map[string]bool{}
		urows, err := tx.QueryContext(ctx, `SELECT key FROM unlocks WHERE player_id = ? AND kind = 'seed'`, PlayerID)
		if err != nil {
			return err
		}
		defer urows.Close()
		for urows.Next() {
			var k string
			if err := urows.Scan(&k); err != nil {
				return err
			}
			unlocked[k] = true
		}
		if err := urows.Err(); err != nil {
			return err
		}
		st.Seeds = make([]SeedState, 0, len(game.Crops))
		for _, c := range game.Crops {
			st.Seeds = append(st.Seeds, SeedState{c.Key, c.DurationMin, c.Unlock, c.Reward, c.LifeH, unlocked[c.Key]})
		}
		a, err := loadActive(ctx, tx)
		if err != nil || a == nil {
			return err
		}
		ps := &PomodoroState{ID: a.ID, PlantType: a.Plant, Status: a.P.Status, PlannedS: a.P.PlannedS,
			RemainingMs: a.P.Remaining(now).Milliseconds(), StartedAt: fmtTime(a.P.StartedAt)}
		if a.PlotID.Valid {
			ps.PlotID = &a.PlotID.Int64
		}
		if a.P.PausedAt != nil {
			t := fmtTime(*a.P.PausedAt)
			ps.PausedAt = &t
		}
		st.Pomodoro = ps
		return nil
	})
	return st, err
}

func strPtr(n sql.NullString) *string {
	if !n.Valid {
		return nil
	}
	return &n.String
}
