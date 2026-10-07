package service

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/luk224/pomofarm/backend/internal/game"
)

type DecorItem struct {
	ID   int64  `json:"id"`
	Kind string `json:"kind"`
	X    int    `json:"x"`
	Y    int    `json:"y"`
}

type DecorOffer struct {
	Kind string `json:"kind"`
	Cost int    `json:"cost"`
}

// HatState is the Dog's straw hat: it can be bought only once the Dog is owned.
type HatState struct {
	Owned     bool `json:"owned"`
	Available bool `json:"available"`
	Cost      int  `json:"cost"`
}

type DecorState struct {
	Items   []DecorItem  `json:"items"`
	Catalog []DecorOffer `json:"catalog"`
	Min     int          `json:"min"`
	Max     int          `json:"max"`
	Hat     HatState     `json:"hat"`
	// Blocked are the cells inside the area that cannot be decorated (the field, the Silo, the Dog).
	Blocked [][2]int `json:"blocked"`
}

func decorView(ctx context.Context, tx *sql.Tx) (DecorState, error) {
	d := DecorState{Items: []DecorItem{}, Min: game.DecorMin, Max: game.DecorMax, Catalog: []DecorOffer{}}
	for _, k := range game.DecorKinds {
		d.Catalog = append(d.Catalog, DecorOffer{k, game.DecorCosts[k]})
	}
	d.Blocked = [][2]int{}
	for x := game.DecorMin; x <= game.DecorMax; x++ {
		for y := game.DecorMin; y <= game.DecorMax; y++ {
			if !game.DecorCellFree(x, y) {
				d.Blocked = append(d.Blocked, [2]int{x, y})
			}
		}
	}
	rows, err := tx.QueryContext(ctx, `SELECT id, kind, x, y FROM structures WHERE player_id = ? AND kind IN ('path','fence','lantern') ORDER BY id`, PlayerID)
	if err != nil {
		return d, err
	}
	defer rows.Close()
	for rows.Next() {
		var it DecorItem
		if err := rows.Scan(&it.ID, &it.Kind, &it.X, &it.Y); err != nil {
			return d, err
		}
		d.Items = append(d.Items, it)
	}
	if err := rows.Err(); err != nil {
		return d, err
	}
	d.Hat.Cost = game.HatCost
	if d.Hat.Owned, err = hasStructure(ctx, tx, "hat"); err != nil {
		return d, err
	}
	dog, err := hasDog(ctx, tx)
	d.Hat.Available = dog && !d.Hat.Owned
	return d, err
}

func hasStructure(ctx context.Context, tx *sql.Tx, kind string) (bool, error) {
	var n int
	err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM structures WHERE player_id = ? AND kind = ?`, PlayerID, kind).Scan(&n)
	return n > 0, err
}

func decorAt(ctx context.Context, tx *sql.Tx, x, y int) (bool, error) {
	var n int
	err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM structures WHERE player_id = ? AND kind IN ('path','fence','lantern') AND x = ? AND y = ?`, PlayerID, x, y).Scan(&n)
	return n > 0, err
}

// BuyDecor buys a piece and puts it on a free background cell. Production is irrelevant to it, so it does not settle.
func (s *Service) BuyDecor(ctx context.Context, kind string, x, y int) error {
	cost, ok := game.DecorCosts[kind]
	if !ok {
		return ErrInvalid
	}
	if !game.DecorCellFree(x, y) {
		return ErrBadCell
	}
	return s.withTx(ctx, func(tx *sql.Tx, now time.Time) error {
		if taken, err := decorAt(ctx, tx, x, y); err != nil {
			return err
		} else if taken {
			return ErrCellTaken
		}
		if err := spendCoins(ctx, tx, cost); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO structures (player_id, kind, x, y) VALUES (?, ?, ?, ?)`, PlayerID, kind, x, y)
		return err
	})
}

// MoveDecor relocates a piece for free.
func (s *Service) MoveDecor(ctx context.Context, id int64, x, y int) error {
	if !game.DecorCellFree(x, y) {
		return ErrBadCell
	}
	return s.withTx(ctx, func(tx *sql.Tx, now time.Time) error {
		var hx, hy int
		err := tx.QueryRowContext(ctx, `SELECT x, y FROM structures WHERE id = ? AND player_id = ? AND kind IN ('path','fence','lantern')`, id, PlayerID).Scan(&hx, &hy)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if hx == x && hy == y {
			return ErrWrongState
		}
		if taken, err := decorAt(ctx, tx, x, y); err != nil {
			return err
		} else if taken {
			return ErrCellTaken
		}
		_, err = tx.ExecContext(ctx, `UPDATE structures SET x = ?, y = ? WHERE id = ?`, x, y, id)
		return err
	})
}

// RemoveDecor takes a piece off the farm. There is no refund: it is the player's own choice, and a piece can simply be moved.
func (s *Service) RemoveDecor(ctx context.Context, id int64) error {
	return s.withTx(ctx, func(tx *sql.Tx, now time.Time) error {
		res, err := tx.ExecContext(ctx, `DELETE FROM structures WHERE id = ? AND player_id = ? AND kind IN ('path','fence','lantern')`, id, PlayerID)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n != 1 {
			return ErrNotFound
		}
		return nil
	})
}

// BuyHat buys the Dog's straw hat (600 🪙). Needs the Dog.
func (s *Service) BuyHat(ctx context.Context) error {
	return s.withTx(ctx, func(tx *sql.Tx, now time.Time) error {
		if dog, err := hasDog(ctx, tx); err != nil {
			return err
		} else if !dog {
			return ErrAnimalLocked
		}
		if owned, err := hasStructure(ctx, tx, "hat"); err != nil {
			return err
		} else if owned {
			return ErrAlreadyOwned
		}
		if err := spendCoins(ctx, tx, game.HatCost); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO structures (player_id, kind, x, y) VALUES (?, 'hat', 0, 0)`, PlayerID)
		return err
	})
}
