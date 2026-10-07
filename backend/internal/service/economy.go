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

	rows, err := tx.QueryContext(ctx, `SELECT id, grow_s, life_s, collected_to, wilts_at FROM plots
		WHERE player_id = ? AND matured_at IS NOT NULL AND collected_to IS NOT NULL AND wilts_at IS NOT NULL
		AND grow_s > 0 AND life_s > 0`, PlayerID)
	if err != nil {
		return err
	}
	defer rows.Close()
	type counted struct {
		id int64
		to time.Time
	}
	var producers []game.Producer
	var advance []counted
	mult := prestigeMultiplier(season)
	for rows.Next() {
		var id, growS, lifeS int64
		var collected, wilts string
		if err := rows.Scan(&id, &growS, &lifeS, &collected, &wilts); err != nil {
			return err
		}
		from, to := parseTime(collected), parseTime(wilts)
		if now.Before(to) {
			to = now
		}
		if !to.After(from) {
			continue
		}
		producers = append(producers, game.Producer{MicroPerHour: game.ProducerMicroPerHour(growS, lifeS, mult), From: from, To: to})
		advance = append(advance, counted{id, to})
	}
	if err := rows.Err(); err != nil {
		return err
	}
	rows.Close()

	if len(producers) > 0 {
		stock = stock.Accrue(producers, game.SiloCapacityHours(level, dog))
		if _, err := tx.ExecContext(ctx, `UPDATE players SET silo_micro = ?, silo_peak_micro_h = ? WHERE id = ?`,
			stock.Micro, stock.PeakMicroPerHour, PlayerID); err != nil {
			return err
		}
		for _, a := range advance {
			if _, err := tx.ExecContext(ctx, `UPDATE plots SET collected_to = ? WHERE id = ?`, fmtTime(a.to), a.id); err != nil {
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
	rows, err := tx.QueryContext(ctx, `SELECT grow_s, life_s, wilts_at FROM plots WHERE player_id = ? AND matured_at IS NOT NULL
		AND wilts_at IS NOT NULL AND grow_s > 0 AND life_s > 0`, PlayerID)
	if err != nil {
		return SiloState{}, err
	}
	defer rows.Close()
	var rate float64
	for rows.Next() {
		var growS, lifeS int64
		var wilts string
		if err := rows.Scan(&growS, &lifeS, &wilts); err != nil {
			return SiloState{}, err
		}
		if parseTime(wilts).After(now) {
			rate += game.ProducerMicroPerHour(growS, lifeS, prestigeMultiplier(seas))
		}
	}
	if err := rows.Err(); err != nil {
		return SiloState{}, err
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
