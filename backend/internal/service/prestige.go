package service

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"time"

	"github.com/luk224/pomofarm/backend/internal/game"
)

// Prestige, "Las Estaciones" (GDD §4.9). When the farm is complete the player may start a new season: plants, structures
// (hives, the Dog, decoration) and 🪙 start over, and in return the biome changes and every 🪙 earned is worth +10% more for good.
// 💧, unlocks (seeds, animals), plots, the Silo level and the whole Harvest Book are kept.

type PrestigeReq struct {
	Key  string `json:"key"` // plots | silo | hives | dog
	Have int    `json:"have"`
	Need int    `json:"need"`
	Met  bool   `json:"met"`
}

type PrestigeState struct {
	Season       int           `json:"season"`
	NextSeason   int           `json:"next_season"`
	NextBiome    string        `json:"next_biome"`
	BonusPct     int           `json:"bonus_pct"`      // what the farm earns above the cap now (0 in the first season)
	NextBonusPct int           `json:"next_bonus_pct"` // …and after the new season
	Requirements []PrestigeReq `json:"requirements"`
	Ready        bool          `json:"ready"`
	// WouldLoseCoinsMilli is what starts over: the balance plus what waits in the Silo.
	WouldLoseCoinsMilli int64 `json:"would_lose_coins_milli"`
	// Unharvested plants are harvested for the player first, so no 💧 is lost.
	Unharvested int `json:"unharvested"`
}

func countInt(ctx context.Context, tx *sql.Tx, q string, args ...any) (int, error) {
	var n int
	err := tx.QueryRowContext(ctx, q, args...).Scan(&n)
	return n, err
}

func prestigeView(ctx context.Context, tx *sql.Tx, season, siloLevel int, coinsAndSiloMilli int64) (PrestigeState, error) {
	plots, err := countInt(ctx, tx, `SELECT COUNT(*) FROM plots WHERE player_id = ?`, PlayerID)
	if err != nil {
		return PrestigeState{}, err
	}
	hives, err := countInt(ctx, tx, `SELECT COUNT(*) FROM structures WHERE player_id = ? AND kind = 'hive'`, PlayerID)
	if err != nil {
		return PrestigeState{}, err
	}
	dog, err := hasDog(ctx, tx)
	if err != nil {
		return PrestigeState{}, err
	}
	dogN := 0
	if dog {
		dogN = 1
	}
	unharvested, err := countInt(ctx, tx, `SELECT COUNT(*) FROM plots WHERE player_id = ? AND state IN ('mature','withered') AND harvested = 0`, PlayerID)
	if err != nil {
		return PrestigeState{}, err
	}
	reqs := []PrestigeReq{
		{Key: "plots", Have: plots, Need: game.PrestigePlots},
		{Key: "silo", Have: siloLevel, Need: game.PrestigeSilo},
		{Key: "hives", Have: hives, Need: game.PrestigeHives},
		{Key: "dog", Have: dogN, Need: 1},
	}
	ready := true
	for i := range reqs {
		reqs[i].Met = reqs[i].Have >= reqs[i].Need
		ready = ready && reqs[i].Met
	}
	if season < 1 {
		season = 1
	}
	pct := int(math.Round(game.PrestigeBonus * 100))
	return PrestigeState{
		Season: season, NextSeason: season + 1, NextBiome: game.BiomeForSeason(season + 1),
		BonusPct: pct * (season - 1), NextBonusPct: pct * season,
		Requirements: reqs, Ready: ready, WouldLoseCoinsMilli: coinsAndSiloMilli, Unharvested: unharvested,
	}, nil
}

// Prestige starts the next season. It needs `confirm` (it cannot be undone), a complete farm and no Pomodoro in progress.
// Plants that are ready but not yet harvested are harvested first, so nothing the player had is lost.
func (s *Service) Prestige(ctx context.Context, confirm bool) error {
	return s.withTx(ctx, func(tx *sql.Tx, now time.Time) error {
		if err := s.settle(ctx, tx, now); err != nil {
			return err
		}
		var season, siloLevel int
		var version int64
		err := tx.QueryRowContext(ctx, `SELECT season, silo_level, version FROM players WHERE id = ?`, PlayerID).Scan(&season, &siloLevel, &version)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNoPlayer
		}
		if err != nil {
			return err
		}
		view, err := prestigeView(ctx, tx, season, siloLevel, 0)
		if err != nil {
			return err
		}
		if !view.Ready {
			return ErrNotReady
		}
		if a, err := loadActive(ctx, tx); err != nil {
			return err
		} else if a != nil {
			return ErrPomodoroActive
		}
		if !confirm {
			return ErrNeedsConfirmation
		}
		rows, err := tx.QueryContext(ctx, `SELECT id FROM plots WHERE player_id = ? AND state IN ('mature','withered') AND harvested = 0 ORDER BY id`, PlayerID)
		if err != nil {
			return err
		}
		var pending []int64
		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return err
			}
			pending = append(pending, id)
		}
		rows.Close()
		for _, id := range pending {
			if _, _, _, err := harvestPlot(ctx, tx, id); err != nil {
				return err
			}
		}
		for _, q := range []string{
			`DELETE FROM structures WHERE player_id = ?`,
			`UPDATE plots SET state = 'empty', plant_type = NULL, planted_at = NULL, grow_s = NULL, matured_at = NULL, harvested = 0,
				life_s = NULL, wilts_at = NULL, collected_to = NULL, version = version + 1 WHERE player_id = ?`,
		} {
			if _, err := tx.ExecContext(ctx, q, PlayerID); err != nil {
				return err
			}
		}
		// harvesting bumped the player's version: read it again for the guarded update
		if err := tx.QueryRowContext(ctx, `SELECT version FROM players WHERE id = ?`, PlayerID).Scan(&version); err != nil {
			return err
		}
		upd, err := tx.ExecContext(ctx, `UPDATE players SET season = season + 1, biome = ?, coins_milli = 0, silo_micro = 0, silo_peak_micro_h = 0,
			last_seen_at = ?, version = version + 1 WHERE id = ? AND version = ?`, game.BiomeForSeason(season+1), fmtTime(now), PlayerID, version)
		if err != nil {
			return err
		}
		if n, _ := upd.RowsAffected(); n != 1 {
			return ErrConflict
		}
		return endRest(ctx, tx)
	})
}
