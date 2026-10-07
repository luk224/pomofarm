package service

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/luk224/pomofarm/backend/internal/game"
)

// ShopState tells the client what can be bought and for how much, so it never hardcodes prices.
type ShopState struct {
	PlotsOwned  int        `json:"plots_owned"`
	PlotsMax    int        `json:"plots_max"`
	NextPlot    *PlotOffer `json:"next_plot"`
	SiloUpgrade *SiloOffer `json:"silo_upgrade"`
}

type PlotOffer struct {
	Number int `json:"number"`
	Cost   int `json:"cost"`
	X      int `json:"x"`
	Y      int `json:"y"`
}

type SiloOffer struct {
	Level         int `json:"level"`
	Cost          int `json:"cost"`
	CapacityHours int `json:"capacity_hours"`
}

func shopView(ctx context.Context, tx *sql.Tx) (ShopState, error) {
	var owned, level int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM plots WHERE player_id = ?`, PlayerID).Scan(&owned); err != nil {
		return ShopState{}, err
	}
	if err := tx.QueryRowContext(ctx, `SELECT silo_level FROM players WHERE id = ?`, PlayerID).Scan(&level); err != nil {
		return ShopState{}, err
	}
	sh := ShopState{PlotsOwned: owned, PlotsMax: game.MaxPlots}
	if x, y, ok := game.PlotPosition(owned + 1); ok {
		sh.NextPlot = &PlotOffer{Number: owned + 1, Cost: game.PlotCost(owned + 1), X: x, Y: y}
	}
	if level+1 < len(game.Silo) {
		sh.SiloUpgrade = &SiloOffer{Level: level + 1, Cost: game.Silo[level+1].Cost, CapacityHours: game.Silo[level+1].CapacityH}
	}
	return sh, nil
}

// spendFocus takes `cost` 💧 from the player, atomically with the purchase that follows (same transaction).
func spendFocus(ctx context.Context, tx *sql.Tx, cost int) error {
	var focus, version int64
	err := tx.QueryRowContext(ctx, `SELECT focus_points, version FROM players WHERE id = ?`, PlayerID).Scan(&focus, &version)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNoPlayer
	}
	if err != nil {
		return err
	}
	if focus < int64(cost) {
		return ErrInsufficientFocus
	}
	upd, err := tx.ExecContext(ctx, `UPDATE players SET focus_points = focus_points - ?, version = version + 1 WHERE id = ? AND version = ?`, cost, PlayerID, version)
	if err != nil {
		return err
	}
	if n, _ := upd.RowsAffected(); n != 1 {
		return ErrConflict
	}
	return nil
}

// BuyPlot buys the next plot (GDD §4.4: plot n costs ceil(3·1.4^(n−2)) 💧). The server decides where it goes.
func (s *Service) BuyPlot(ctx context.Context) error {
	return s.withTx(ctx, func(tx *sql.Tx, now time.Time) error {
		if err := s.settle(ctx, tx, now); err != nil {
			return err
		}
		var owned int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM plots WHERE player_id = ?`, PlayerID).Scan(&owned); err != nil {
			return err
		}
		x, y, ok := game.PlotPosition(owned + 1)
		if !ok {
			return ErrMaxedOut
		}
		if err := spendFocus(ctx, tx, game.PlotCost(owned+1)); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO plots (player_id, x, y) VALUES (?, ?, ?)`, PlayerID, x, y)
		return err
	})
}

// UpgradeSilo buys the next Silo level (GDD §4.5).
func (s *Service) UpgradeSilo(ctx context.Context) error {
	return s.withTx(ctx, func(tx *sql.Tx, now time.Time) error {
		if err := s.settle(ctx, tx, now); err != nil {
			return err
		}
		var level int
		if err := tx.QueryRowContext(ctx, `SELECT silo_level FROM players WHERE id = ?`, PlayerID).Scan(&level); errors.Is(err, sql.ErrNoRows) {
			return ErrNoPlayer
		} else if err != nil {
			return err
		}
		if level+1 >= len(game.Silo) {
			return ErrMaxedOut
		}
		if err := spendFocus(ctx, tx, game.Silo[level+1].Cost); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `UPDATE players SET silo_level = ? WHERE id = ?`, level+1, PlayerID)
		return err
	})
}
