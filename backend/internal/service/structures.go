package service

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/luk224/pomofarm/backend/internal/game"
)

// ---------- what the client sees ----------

type HiveState struct {
	ID     int64 `json:"id"`
	PlotID int64 `json:"plot_id"`
	X      int   `json:"x"`
	Y      int   `json:"y"`
}

// BeesState: unlocked with 💧, then up to four hives bought with 🪙 (GDD §4.7). Costs are whole 🪙.
type BeesState struct {
	Unlocked   bool        `json:"unlocked"`
	UnlockCost int         `json:"unlock_cost"`
	Hives      []HiveState `json:"hives"`
	Max        int         `json:"max"`
	NextCost   *int        `json:"next_cost"` // nil once all four are bought
}

type DogState struct {
	Unlocked       bool `json:"unlocked"`
	UnlockCost     int  `json:"unlock_cost"`
	Owned          bool `json:"owned"`
	Cost           int  `json:"cost"`
	SiloBonusHours int  `json:"silo_bonus_hours"`
}

type AutomationState struct {
	Bees BeesState `json:"bees"`
	Dog  DogState  `json:"dog"`
}

func animalUnlocked(ctx context.Context, tx *sql.Tx, key string) (bool, error) {
	var n int
	err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM unlocks WHERE player_id = ? AND kind = 'animal' AND key = ?`, PlayerID, key).Scan(&n)
	return n > 0, err
}

func automationView(ctx context.Context, tx *sql.Tx) (AutomationState, error) {
	var a AutomationState
	var err error
	a.Bees = BeesState{UnlockCost: game.UnlockBees, Max: game.MaxHives, Hives: []HiveState{}}
	if a.Bees.Unlocked, err = animalUnlocked(ctx, tx, "bees"); err != nil {
		return a, err
	}
	rows, err := tx.QueryContext(ctx, `SELECT s.id, COALESCE(p.id, 0), s.x, s.y FROM structures s
		LEFT JOIN plots p ON p.player_id = s.player_id AND p.x = s.x AND p.y = s.y
		WHERE s.player_id = ? AND s.kind = 'hive' ORDER BY s.id`, PlayerID)
	if err != nil {
		return a, err
	}
	defer rows.Close()
	for rows.Next() {
		var h HiveState
		if err := rows.Scan(&h.ID, &h.PlotID, &h.X, &h.Y); err != nil {
			return a, err
		}
		a.Bees.Hives = append(a.Bees.Hives, h)
	}
	if err := rows.Err(); err != nil {
		return a, err
	}
	if n := len(a.Bees.Hives); n < game.MaxHives {
		c := game.HiveCost(n)
		a.Bees.NextCost = &c
	}
	a.Dog = DogState{UnlockCost: game.UnlockDog, Cost: game.DogCost, SiloBonusHours: game.DogSiloBonus}
	if a.Dog.Unlocked, err = animalUnlocked(ctx, tx, "dog"); err != nil {
		return a, err
	}
	a.Dog.Owned, err = hasDog(ctx, tx)
	return a, err
}

// ---------- unlocking animals (💧, a one-off unlock like seeds, GDD §4.1) ----------

var animalUnlockCost = map[string]int{"bees": game.UnlockBees, "dog": game.UnlockDog}

// UnlockAnimal buys the right to buy Bees or the Dog with 💧.
func (s *Service) UnlockAnimal(ctx context.Context, key string) error {
	cost, ok := animalUnlockCost[key]
	if !ok {
		return ErrInvalid
	}
	return s.withTx(ctx, func(tx *sql.Tx, now time.Time) error {
		if err := s.settle(ctx, tx, now); err != nil {
			return err
		}
		have, err := animalUnlocked(ctx, tx, key)
		if err != nil {
			return err
		}
		if have {
			return ErrAlreadyUnlocked
		}
		if err := spendFocus(ctx, tx, cost); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO unlocks (player_id, kind, key, at) VALUES (?, 'animal', ?, ?)`, PlayerID, key, fmtTime(now))
		return err
	})
}

// ---------- spending 🪙 ----------

// spendCoins takes whole 🪙 from the balance in the same transaction as the purchase it pays for.
func spendCoins(ctx context.Context, tx *sql.Tx, coins int) error {
	var have, version int64
	err := tx.QueryRowContext(ctx, `SELECT coins_milli, version FROM players WHERE id = ?`, PlayerID).Scan(&have, &version)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNoPlayer
	}
	if err != nil {
		return err
	}
	cost := int64(coins) * game.MilliPerCoin
	if have < cost {
		return ErrInsufficientCoins
	}
	upd, err := tx.ExecContext(ctx, `UPDATE players SET coins_milli = coins_milli - ?, version = version + 1 WHERE id = ? AND version = ?`, cost, PlayerID, version)
	if err != nil {
		return err
	}
	if n, _ := upd.RowsAffected(); n != 1 {
		return ErrConflict
	}
	return nil
}

// plotCell returns the grid cell of a plot the player owns.
func plotCell(ctx context.Context, tx *sql.Tx, plotID int64) (x, y int, err error) {
	err = tx.QueryRowContext(ctx, `SELECT x, y FROM plots WHERE id = ? AND player_id = ?`, plotID, PlayerID).Scan(&x, &y)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, 0, ErrNotFound
	}
	return x, y, err
}

func hiveAt(ctx context.Context, tx *sql.Tx, x, y int) (bool, error) {
	var n int
	err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM structures WHERE player_id = ? AND kind = 'hive' AND x = ? AND y = ?`, PlayerID, x, y).Scan(&n)
	return n > 0, err
}

// BuyHive buys the next hive (4.000 / 6.000 / 9.000 / 13.500 🪙) and puts it beside the given plot, without taking its
// place. Production is settled first (every request does), so the +25% starts exactly now.
func (s *Service) BuyHive(ctx context.Context, plotID int64) error {
	return s.withTx(ctx, func(tx *sql.Tx, now time.Time) error {
		if err := s.settle(ctx, tx, now); err != nil {
			return err
		}
		if ok, err := animalUnlocked(ctx, tx, "bees"); err != nil {
			return err
		} else if !ok {
			return ErrAnimalLocked
		}
		var owned int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM structures WHERE player_id = ? AND kind = 'hive'`, PlayerID).Scan(&owned); err != nil {
			return err
		}
		if owned >= game.MaxHives {
			return ErrMaxedOut
		}
		x, y, err := plotCell(ctx, tx, plotID)
		if err != nil {
			return err
		}
		if taken, err := hiveAt(ctx, tx, x, y); err != nil {
			return err
		} else if taken {
			return ErrCellTaken
		}
		if err := spendCoins(ctx, tx, game.HiveCost(owned)); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO structures (player_id, kind, x, y) VALUES (?, 'hive', ?, ?)`, PlayerID, x, y)
		return err
	})
}

// MoveHive relocates a hive to another plot for free (a wrong choice must not cost thousands of 🪙: GDD §1.1,
// forgiving model). Production is settled first, so the old area keeps its bonus until now and the new one gets it from now.
func (s *Service) MoveHive(ctx context.Context, hiveID, plotID int64) error {
	return s.withTx(ctx, func(tx *sql.Tx, now time.Time) error {
		if err := s.settle(ctx, tx, now); err != nil {
			return err
		}
		var hx, hy int
		err := tx.QueryRowContext(ctx, `SELECT x, y FROM structures WHERE id = ? AND player_id = ? AND kind = 'hive'`, hiveID, PlayerID).Scan(&hx, &hy)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		x, y, err := plotCell(ctx, tx, plotID)
		if err != nil {
			return err
		}
		if x == hx && y == hy {
			return ErrWrongState // already there
		}
		if taken, err := hiveAt(ctx, tx, x, y); err != nil {
			return err
		} else if taken {
			return ErrCellTaken
		}
		_, err = tx.ExecContext(ctx, `UPDATE structures SET x = ?, y = ? WHERE id = ?`, x, y, hiveID)
		return err
	})
}

// BuyDog buys the Pet Dog (30.000 🪙): it empties the Silo by itself and adds 12 h of capacity (GDD §4.7).
func (s *Service) BuyDog(ctx context.Context) error {
	return s.withTx(ctx, func(tx *sql.Tx, now time.Time) error {
		if err := s.settle(ctx, tx, now); err != nil {
			return err
		}
		if ok, err := animalUnlocked(ctx, tx, "dog"); err != nil {
			return err
		} else if !ok {
			return ErrAnimalLocked
		}
		if has, err := hasDog(ctx, tx); err != nil {
			return err
		} else if has {
			return ErrAlreadyOwned
		}
		if err := spendCoins(ctx, tx, game.DogCost); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO structures (player_id, kind, x, y) VALUES (?, 'dog', 0, 0)`, PlayerID)
		return err
	})
}
