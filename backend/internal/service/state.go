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
	Biome         string `json:"biome"`
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
	// Bonus is set only while the plant is producing: what its neighbours and patterns earn it.
	Bonus *PlotBonus `json:"bonus"`
}

type PlotBonus struct {
	Multiplier float64 `json:"multiplier"` // total applied to its production (synergies × prestige)
	Neighbours int     `json:"neighbours"` // compatible orthogonal neighbours
	Garden     bool    `json:"garden"`     // inside a Huerto completo
	Bees       bool    `json:"bees"`       // inside the 3×3 area of a hive
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
	Tag         *string `json:"tag"`
	// Strict mode (GDD §3.3): chosen when it started. Pauses so far and the time spent paused (including the pause in
	// progress, as of this answer), so the client can show "1 of 2 pauses" without ever blocking anything.
	Strict   bool  `json:"strict"`
	Pauses   int   `json:"pauses"`
	PausedMs int64 `json:"paused_ms"`
}

// SeedState lets the client list seeds and their prices without knowing the balance.
type SeedState struct {
	Key         string `json:"key"`
	DurationMin int    `json:"duration_min"`
	UnlockCost  int    `json:"unlock_cost"`
	Reward      int    `json:"reward"`
	LifeH       int    `json:"life_h"`
	Unlocked    bool   `json:"unlocked"`
	// Compatible are the two crops this one combines with (the ring of GDD §4.6).
	Compatible []string `json:"compatible"`
}

// State is everything the client needs to draw the game. The client displays it
// and never computes economy or time itself.
type State struct {
	ServerTime string            `json:"server_time"`
	Player     PlayerState       `json:"player"`
	Plots      []PlotState       `json:"plots"`
	Seeds      []SeedState       `json:"seeds"`
	Pomodoro   *PomodoroState    `json:"pomodoro"`
	Silo       SiloState         `json:"silo"`
	Shop       ShopState         `json:"shop"`
	Rest       *RestState        `json:"rest"`
	Automation AutomationState   `json:"automation"`
	Decor      DecorState        `json:"decor"`
	Prestige   PrestigeState     `json:"prestige"`
	RecentTags []string          `json:"recent_tags"`
	Settings   map[string]string `json:"settings"`
}

func (s *Service) State(ctx context.Context) (State, error) {
	var st State
	err := s.withTx(ctx, func(tx *sql.Tx, now time.Time) error {
		if err := s.settle(ctx, tx, now); err != nil {
			return err
		}
		st.ServerTime = fmtTime(now)
		err := tx.QueryRowContext(ctx, `SELECT name, focus_points, lifetime_focus, coins_milli, silo_level, season, biome FROM players WHERE id = ?`, PlayerID).
			Scan(&st.Player.Name, &st.Player.FocusPoints, &st.Player.LifetimeFocus, &st.Player.CoinsMilli, &st.Player.SiloLevel, &st.Player.Season, &st.Player.Biome)
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
			st.Seeds = append(st.Seeds, SeedState{c.Key, c.DurationMin, c.Unlock, c.Reward, c.LifeH, unlocked[c.Key], game.CompatibleWith(c.Key)})
		}
		silo, err := siloView(ctx, tx, now)
		if err != nil {
			return err
		}
		st.Silo = silo
		shop, err := shopView(ctx, tx)
		if err != nil {
			return err
		}
		st.Shop = shop
		rest, err := restView(ctx, tx, now)
		if err != nil {
			return err
		}
		st.Rest = rest
		st.RecentTags = []string{}
		trows, err := tx.QueryContext(ctx, `SELECT t.name FROM tags t JOIN pomodoros p ON p.tag_id = t.id
			WHERE t.player_id = ? GROUP BY t.id ORDER BY MAX(p.id) DESC LIMIT 8`, PlayerID)
		if err != nil {
			return err
		}
		defer trows.Close()
		for trows.Next() {
			var n string
			if err := trows.Scan(&n); err != nil {
				return err
			}
			st.RecentTags = append(st.RecentTags, n)
		}
		if err := trows.Err(); err != nil {
			return err
		}
		st.Settings = map[string]string{}
		srows, err := tx.QueryContext(ctx, `SELECT key, value FROM settings WHERE player_id = ?`, PlayerID)
		if err != nil {
			return err
		}
		defer srows.Close()
		for srows.Next() {
			var k, v string
			if err := srows.Scan(&k, &v); err != nil {
				return err
			}
			st.Settings[k] = v
		}
		if err := srows.Err(); err != nil {
			return err
		}
		farm, err := loadFarm(ctx, tx)
		if err != nil {
			return err
		}
		bees, err := beeCells(ctx, tx)
		if err != nil {
			return err
		}
		bonuses := game.CurrentBonuses(farm, now, bees)
		prestige := prestigeMultiplier(st.Player.Season)
		for i := range st.Plots {
			if b, ok := bonuses[st.Plots[i].ID]; ok {
				st.Plots[i].Bonus = &PlotBonus{Multiplier: b.Total * prestige, Neighbours: b.Neighbours, Garden: b.Garden, Bees: b.Bees}
			}
		}
		automation, err := automationView(ctx, tx)
		if err != nil {
			return err
		}
		st.Automation = automation
		if st.Decor, err = decorView(ctx, tx); err != nil {
			return err
		}
		if st.Prestige, err = prestigeView(ctx, tx, st.Player.Season, st.Player.SiloLevel, st.Player.CoinsMilli+st.Silo.ContentMilli); err != nil {
			return err
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
		var tag sql.NullString
		if err := tx.QueryRowContext(ctx, `SELECT t.name FROM pomodoros p LEFT JOIN tags t ON t.id = p.tag_id WHERE p.id = ?`, a.ID).Scan(&tag); err != nil {
			return err
		}
		ps.Tag = strPtr(tag)
		if err := tx.QueryRowContext(ctx, `SELECT strict, (SELECT COUNT(*) FROM pomodoro_events WHERE pomodoro_id = ? AND kind = 'pause') FROM pomodoros WHERE id = ?`, a.ID, a.ID).
			Scan(&ps.Strict, &ps.Pauses); err != nil {
			return err
		}
		ps.PausedMs = a.P.PausedTotalS * 1000
		if a.P.PausedAt != nil {
			ps.PausedMs += now.Sub(*a.P.PausedAt).Milliseconds()
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
