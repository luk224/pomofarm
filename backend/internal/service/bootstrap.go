package service

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/luk224/pomofarm/backend/internal/game"
)

// EnsurePlayer creates the single player on first run: one empty plot and the
// free Daisy seed unlocked (GDD §2.1). It does nothing if the player exists.
func (s *Service) EnsurePlayer(ctx context.Context, name string) error {
	return s.withTx(ctx, func(tx *sql.Tx, now time.Time) error {
		var id int64
		err := tx.QueryRowContext(ctx, `SELECT id FROM players WHERE id = ?`, PlayerID).Scan(&id)
		if err == nil {
			return nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		t := fmtTime(now)
		if _, err := tx.ExecContext(ctx, `INSERT INTO players (id, name, created_at, last_seen_at) VALUES (?, ?, ?, ?)`,
			PlayerID, name, t, t); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO plots (player_id, x, y) VALUES (?, 0, 0)`, PlayerID); err != nil {
			return err
		}
		for _, c := range game.Crops {
			if c.Unlock == 0 {
				if _, err := tx.ExecContext(ctx, `INSERT INTO unlocks (player_id, kind, key, at) VALUES (?, 'seed', ?, ?)`,
					PlayerID, c.Key, t); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// UnlockSeed buys a seed for good with 💧 (GDD §4.1: a one-off unlock, not a cost per planting).
func (s *Service) UnlockSeed(ctx context.Context, key string) error {
	crop, ok := game.CropByKey(key)
	if !ok {
		return ErrInvalid
	}
	return s.withTx(ctx, func(tx *sql.Tx, now time.Time) error {
		var focus, version int64
		err := tx.QueryRowContext(ctx, `SELECT focus_points, version FROM players WHERE id = ?`, PlayerID).Scan(&focus, &version)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNoPlayer
		}
		if err != nil {
			return err
		}
		var n int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM unlocks WHERE player_id = ? AND kind = 'seed' AND key = ?`,
			PlayerID, key).Scan(&n); err != nil {
			return err
		}
		if n > 0 {
			return ErrAlreadyUnlocked
		}
		if focus < int64(crop.Unlock) {
			return ErrInsufficientFocus
		}
		upd, err := tx.ExecContext(ctx, `UPDATE players SET focus_points = focus_points - ?, version = version + 1
			WHERE id = ? AND version = ?`, crop.Unlock, PlayerID, version)
		if err != nil {
			return err
		}
		if c, _ := upd.RowsAffected(); c != 1 {
			return ErrConflict
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO unlocks (player_id, kind, key, at) VALUES (?, 'seed', ?, ?)`,
			PlayerID, key, fmtTime(now))
		return err
	})
}
