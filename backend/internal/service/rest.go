package service

import (
	"context"
	"database/sql"
	"strconv"
	"time"

	"github.com/luk224/pomofarm/backend/internal/game"
)

const (
	minRestMin = 1
	maxRestMin = 60
)

// RestState is the optional break after a Pomodoro (GDD §3.2). It is informative: nothing is blocked while it runs.
type RestState struct {
	TotalS      int64  `json:"total_s"`
	RemainingMs int64  `json:"remaining_ms"`
	EndsAt      string `json:"ends_at"`
}

func loadSettings(ctx context.Context, tx *sql.Tx) (map[string]string, error) {
	rows, err := tx.QueryContext(ctx, `SELECT key, value FROM settings WHERE player_id = ?`, PlayerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		out[k] = v
	}
	return out, rows.Err()
}

// restMinutes is the configured length for the Pomodoro's bucket, or the GDD default when unset or invalid.
func restMinutes(settings map[string]string, durationMin int) int {
	if v, err := strconv.Atoi(settings["rest_"+game.RestBucket(durationMin)+"_min"]); err == nil && v >= minRestMin && v <= maxRestMin {
		return v
	}
	return game.RestMin(durationMin)
}

// startRest begins the break when the player harvests soon after the Pomodoro ended and has rests switched on.
func startRest(ctx context.Context, tx *sql.Tx, now time.Time, growS int64, matured time.Time) error {
	settings, err := loadSettings(ctx, tx)
	if err != nil {
		return err
	}
	if settings["rest_enabled"] == "0" || matured.IsZero() || now.Sub(matured) > game.RestOfferWindow {
		return nil
	}
	end := now.Add(time.Duration(restMinutes(settings, int(growS/60))) * time.Minute)
	_, err = tx.ExecContext(ctx, `UPDATE players SET rest_started_at = ?, rest_until = ? WHERE id = ?`, fmtTime(now), fmtTime(end), PlayerID)
	return err
}

// endRest clears any running rest (planting a new Pomodoro, or the player skipping it).
func endRest(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `UPDATE players SET rest_started_at = NULL, rest_until = NULL WHERE id = ?`, PlayerID)
	return err
}

func restView(ctx context.Context, tx *sql.Tx, now time.Time) (*RestState, error) {
	var started, until sql.NullString
	if err := tx.QueryRowContext(ctx, `SELECT rest_started_at, rest_until FROM players WHERE id = ?`, PlayerID).Scan(&started, &until); err != nil {
		return nil, err
	}
	if !started.Valid || !until.Valid {
		return nil, nil
	}
	end, begin := parseTime(until.String), parseTime(started.String)
	if !end.After(now) {
		return nil, nil
	}
	return &RestState{TotalS: int64(end.Sub(begin).Seconds()), RemainingMs: end.Sub(now).Milliseconds(), EndsAt: fmtTime(end)}, nil
}

// SkipRest ends the break early.
func (s *Service) SkipRest(ctx context.Context) error {
	return s.withTx(ctx, func(tx *sql.Tx, now time.Time) error {
		if err := s.settle(ctx, tx, now); err != nil {
			return err
		}
		r, err := restView(ctx, tx, now)
		if err != nil {
			return err
		}
		if r == nil {
			return ErrNoRest
		}
		return endRest(ctx, tx)
	})
}

// validSetting checks the value of a setting the client may write.
func validSetting(key, value string) bool {
	switch key {
	case "rest_enabled", "strict_mode":
		return value == "0" || value == "1"
	case "rest_short_min", "rest_medium_min", "rest_long_min":
		n, err := strconv.Atoi(value)
		return err == nil && n >= minRestMin && n <= maxRestMin
	default:
		return len(value) <= 200
	}
}
