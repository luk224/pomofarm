package service

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// ClearPlot removes a standing plant so the plot can be replanted.
//   - empty or still growing: nothing to clear (growing plants are cancelled instead).
//   - mature: only after harvesting (the 💧 reward is never lost), and with confirm=true,
//     because it stops producing (GDD §3.1).
//   - withered: free, no confirmation (also only after harvesting, so the 💧 reward is never lost).
func (s *Service) ClearPlot(ctx context.Context, plotID int64, confirm bool) error {
	return s.withTx(ctx, func(tx *sql.Tx, now time.Time) error {
		if err := s.settle(ctx, tx, now); err != nil {
			return err
		}
		var state string
		var harvested, version int64
		err := tx.QueryRowContext(ctx, `SELECT state, harvested, version FROM plots WHERE id = ? AND player_id = ?`, plotID, PlayerID).
			Scan(&state, &harvested, &version)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		switch state {
		case "empty":
			return ErrWrongState
		case "growing":
			return ErrPlotBusy
		case "mature":
			if harvested == 0 {
				return ErrHarvestFirst
			}
			if !confirm {
				return ErrNeedsConfirmation
			}
		case "withered":
			if harvested == 0 {
				return ErrHarvestFirst // its 💧 reward is still waiting
			}
			// free and without confirmation: it has nothing left to lose (GDD §3.1)
		}
		upd, err := tx.ExecContext(ctx, `UPDATE plots SET state = 'empty', plant_type = NULL, planted_at = NULL, grow_s = NULL,
			matured_at = NULL, harvested = 0, life_s = NULL, wilts_at = NULL, collected_to = NULL, version = version + 1
			WHERE id = ? AND version = ?`, plotID, version)
		if err != nil {
			return err
		}
		if n, _ := upd.RowsAffected(); n != 1 {
			return ErrConflict
		}
		return nil
	})
}

// settingKeys are the only per-player settings the client may write.
var settingKeys = map[string]bool{"tutorial_done": true, "hints_seen": true, "rest_enabled": true, "rest_short_min": true, "rest_medium_min": true, "rest_long_min": true}

// SetSetting stores a small per-player setting (e.g. that the tutorial was dismissed).
func (s *Service) SetSetting(ctx context.Context, key, value string) error {
	if !settingKeys[key] || !validSetting(key, value) {
		return ErrInvalid
	}
	return s.withTx(ctx, func(tx *sql.Tx, now time.Time) error {
		var id int64
		if err := tx.QueryRowContext(ctx, `SELECT id FROM players WHERE id = ?`, PlayerID).Scan(&id); errors.Is(err, sql.ErrNoRows) {
			return ErrNoPlayer
		} else if err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO settings (player_id, key, value) VALUES (?, ?, ?)
			ON CONFLICT(player_id, key) DO UPDATE SET value = excluded.value`, PlayerID, key, value)
		return err
	})
}
