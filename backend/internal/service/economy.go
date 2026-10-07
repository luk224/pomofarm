package service

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/luk224/pomofarm/backend/internal/game"
)

// accrue moves the production of every mature plant, from where it was last counted up to `now`, into the
// Silo (GDD §6.2). It runs at the start of every request, so there are no background jobs: the farm "catches up"
// whenever anyone looks at it. Plants stop producing at wilts_at; a plant's collected_to only moves forward, so a
// clock that went backwards simply yields an empty window.
func (s *Service) accrue(ctx context.Context, tx *sql.Tx, now time.Time) error {
	var (
		stock    game.SiloStock
		level    int
		season   int
		lastSeen string
	)
	err := tx.QueryRowContext(ctx, `SELECT silo_micro, silo_peak_micro_h, silo_level, season, last_seen_at FROM players WHERE id = ?`, PlayerID).
		Scan(&stock.Micro, &stock.PeakMicroPerHour, &level, &season, &lastSeen)
	if errors.Is(err, sql.ErrNoRows) {
		return nil // fresh install: the player is created right after
	}
	if err != nil {
		return err
	}
	dog, err := hasDog(ctx, tx)
	if err != nil {
		return err
	}

	farm, err := loadFarm(ctx, tx)
	if err != nil {
		return err
	}
	// Synergies change whenever a neighbour matures or wilts, so the farm is counted piece by piece (game.FarmProducers).
	bees, err := beeCells(ctx, tx)
	if err != nil {
		return err
	}
	producers := game.FarmProducers(farm, now, prestigeMultiplier(season), bees)
	if len(producers) > 0 {
		stock = stock.Accrue(producers, game.SiloCapacityHours(level, dog))
		if _, err := tx.ExecContext(ctx, `UPDATE players SET silo_micro = ?, silo_peak_micro_h = ? WHERE id = ?`,
			stock.Micro, stock.PeakMicroPerHour, PlayerID); err != nil {
			return err
		}
	}
	for _, p := range farm { // move every plant's marker forward to now, or to its death if it died before
		to := p.Wilts
		if now.Before(to) {
			to = now
		}
		if to.After(p.Collected) {
			if _, err := tx.ExecContext(ctx, `UPDATE plots SET collected_to = ? WHERE id = ?`, fmtTime(to), p.ID); err != nil {
				return err
			}
		}
	}
	if prev := parseTime(lastSeen); now.After(prev) {
		if _, err := tx.ExecContext(ctx, `UPDATE players SET last_seen_at = ? WHERE id = ?`, fmtTime(now), PlayerID); err != nil {
			return err
		}
	}
	if dog && stock.Micro >= 1000 { // the Dog empties the Silo by himself (GDD §4.7)
		_, err = collectSilo(ctx, tx)
	}
	return err
}

// beeCells reads the player's hives and returns the cells their 3×3 areas cover (GDD §4.6, §4.7). Hives only change
// through requests that settle production first, so coverage is constant between two settlements.
func beeCells(ctx context.Context, tx *sql.Tx) (map[[2]int]bool, error) {
	rows, err := tx.QueryContext(ctx, `SELECT x, y FROM structures WHERE player_id = ? AND kind = 'hive'`, PlayerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var hives [][2]int
	for rows.Next() {
		var h [2]int
		if err := rows.Scan(&h[0], &h[1]); err != nil {
			return nil, err
		}
		hives = append(hives, h)
	}
	return game.BeeCoverage(hives), rows.Err()
}

// loadFarm reads every plot that holds (or held) a mature plant, with the times production depends on.
func loadFarm(ctx context.Context, tx *sql.Tx) ([]game.FarmPlot, error) {
	rows, err := tx.QueryContext(ctx, `SELECT id, x, y, COALESCE(plant_type, ''), grow_s, life_s, matured_at, wilts_at, collected_to FROM plots
		WHERE player_id = ? AND matured_at IS NOT NULL AND wilts_at IS NOT NULL AND collected_to IS NOT NULL AND grow_s > 0 AND life_s > 0`, PlayerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var farm []game.FarmPlot
	for rows.Next() {
		var p game.FarmPlot
		var matured, wilts, collected string
		if err := rows.Scan(&p.ID, &p.X, &p.Y, &p.Crop, &p.GrowS, &p.LifeS, &matured, &wilts, &collected); err != nil {
			return nil, err
		}
		p.Matured, p.Wilts, p.Collected = parseTime(matured), parseTime(wilts), parseTime(collected)
		farm = append(farm, p)
	}
	return farm, rows.Err()
}

// prestigeMultiplier: +10% of 🪙 per completed season, applied above the per-plot cap (GDD §4.6, §4.9).
func prestigeMultiplier(season int) float64 {
	if season < 1 {
		season = 1
	}
	return 1 + game.PrestigeBonus*float64(season-1)
}

func hasDog(ctx context.Context, tx *sql.Tx) (bool, error) {
	var n int
	err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM structures WHERE player_id = ? AND kind = 'dog'`, PlayerID).Scan(&n)
	return n > 0, err
}

// collectSilo moves whole thousandths of a coin from the Silo to the player's balance. The sub-thousandth
// remainder stays in the Silo so nothing is lost, and the capacity window restarts.
func collectSilo(ctx context.Context, tx *sql.Tx) (int64, error) {
	var micro int64
	if err := tx.QueryRowContext(ctx, `SELECT silo_micro FROM players WHERE id = ?`, PlayerID).Scan(&micro); err != nil {
		return 0, err
	}
	milli := micro / 1000
	if milli <= 0 {
		return 0, nil
	}
	_, err := tx.ExecContext(ctx, `UPDATE players SET coins_milli = coins_milli + ?, silo_micro = ?, silo_peak_micro_h = 0,
		version = version + 1 WHERE id = ?`, milli, micro%1000, PlayerID)
	return milli, err
}

// CollectSilo is the player emptying the Silo (the Dog does the same automatically).
func (s *Service) CollectSilo(ctx context.Context) (int64, error) {
	var collected int64
	err := s.withTx(ctx, func(tx *sql.Tx, now time.Time) error {
		if err := s.settle(ctx, tx, now); err != nil {
			return err
		}
		var level int
		if err := tx.QueryRowContext(ctx, `SELECT silo_level FROM players WHERE id = ?`, PlayerID).Scan(&level); errors.Is(err, sql.ErrNoRows) {
			return ErrNoPlayer
		} else if err != nil {
			return err
		}
		var err error
		collected, err = collectSilo(ctx, tx)
		if err != nil {
			return err
		}
		if collected == 0 {
			return ErrSiloEmpty
		}
		return nil
	})
	return collected, err
}

// SiloState is what the client draws for the Silo; it only displays it.
type SiloState struct {
	ContentMilli  int64 `json:"content_milli"`
	CapacityMilli int64 `json:"capacity_milli"`
	CapacityHours int   `json:"capacity_hours"`
	// RateMilliPerHour is what the farm produces right now (mature, not wilted plants).
	RateMilliPerHour int64 `json:"rate_milli_per_h"`
	Full             bool  `json:"full"`
}

func siloView(ctx context.Context, tx *sql.Tx, now time.Time) (SiloState, error) {
	var (
		micro, peak int64
		level, seas int
	)
	if err := tx.QueryRowContext(ctx, `SELECT silo_micro, silo_peak_micro_h, silo_level, season FROM players WHERE id = ?`, PlayerID).
		Scan(&micro, &peak, &level, &seas); err != nil {
		return SiloState{}, err
	}
	dog, err := hasDog(ctx, tx)
	if err != nil {
		return SiloState{}, err
	}
	farm, err := loadFarm(ctx, tx)
	if err != nil {
		return SiloState{}, err
	}
	bees, err := beeCells(ctx, tx)
	if err != nil {
		return SiloState{}, err
	}
	var rate float64
	bonuses := game.CurrentBonuses(farm, now, bees)
	for _, p := range farm {
		if b, producing := bonuses[p.ID]; producing {
			rate += game.ProducerMicroPerHour(p.GrowS, p.LifeS, b.Total*prestigeMultiplier(seas))
		}
	}
	hours := game.SiloCapacityHours(level, dog)
	// Capacity follows the highest rate since the last collection, but show at least today's rate.
	shown := float64(peak)
	if rate > shown {
		shown = rate
	}
	capMicro := int64(hours * shown)
	return SiloState{
		ContentMilli:     micro / 1000,
		CapacityMilli:    capMicro / 1000,
		CapacityHours:    int(hours),
		RateMilliPerHour: int64(rate / 1000),
		Full:             capMicro > 0 && micro >= capMicro,
	}, nil
}
