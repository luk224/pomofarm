// Package service orchestrates game rules against the database. All time comes
// from the injected Clock (the server's); the client never supplies timestamps.
package service

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/luk224/pomofarm/backend/internal/game"
)

// PlayerID is the only player (one per installation, GDD §6.4).
const PlayerID = 1

type Service struct {
	DB    *sql.DB
	Clock game.Clock
}

func New(db *sql.DB, clock game.Clock) *Service { return &Service{DB: db, Clock: clock} }

const timeLayout = time.RFC3339Nano

func fmtTime(t time.Time) string { return t.UTC().Format(timeLayout) }

func parseTime(s string) time.Time {
	t, _ := time.Parse(timeLayout, s)
	return t
}

func parseNullTime(s sql.NullString) *time.Time {
	if !s.Valid {
		return nil
	}
	t := parseTime(s.String)
	return &t
}

func nullTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return fmtTime(*t)
}

// withTx runs fn in a transaction that is committed only if fn succeeds.
func (s *Service) withTx(ctx context.Context, fn func(tx *sql.Tx, now time.Time) error) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := fn(tx, s.Clock.Now().UTC()); err != nil {
		return err
	}
	return tx.Commit()
}

func isUnique(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}

// activeRow is the running/paused Pomodoro as stored.
type activeRow struct {
	ID     int64
	PlotID sql.NullInt64
	Plant  string
	P      game.Pomodoro
}

func loadActive(ctx context.Context, tx *sql.Tx) (*activeRow, error) {
	var (
		a       activeRow
		started string
		paused  sql.NullString
	)
	err := tx.QueryRowContext(ctx, `SELECT id, plot_id, plant_type, planned_s, started_at, paused_at, paused_total_s, status
		FROM pomodoros WHERE player_id = ? AND status IN ('running','paused')`, PlayerID).
		Scan(&a.ID, &a.PlotID, &a.Plant, &a.P.PlannedS, &started, &paused, &a.P.PausedTotalS, &a.P.Status)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	a.P.StartedAt = parseTime(started)
	a.P.PausedAt = parseNullTime(paused)
	return &a, nil
}

func saveActive(ctx context.Context, tx *sql.Tx, a *activeRow) error {
	_, err := tx.ExecContext(ctx, `UPDATE pomodoros SET status = ?, paused_at = ?, paused_total_s = ?, ended_at = ? WHERE id = ?`,
		a.P.Status, nullTime(a.P.PausedAt), a.P.PausedTotalS, nullTime(a.P.EndedAt), a.ID)
	return err
}

func addEvent(ctx context.Context, tx *sql.Tx, pomodoroID int64, kind string, at time.Time) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO pomodoro_events (pomodoro_id, kind, at) VALUES (?, ?, ?)`, pomodoroID, kind, fmtTime(at))
	return err
}

// settle completes the active Pomodoro if its time is up, turning its plot
// into a mature plant. Called at the start of every request: there are no
// background jobs (GDD §6.2).
func (s *Service) settle(ctx context.Context, tx *sql.Tx, now time.Time) error {
	a, err := loadActive(ctx, tx)
	if err != nil {
		return err
	}
	if a != nil && a.P.Finished(now) {
		if err := s.completeActive(ctx, tx, a, now); err != nil {
			return err
		}
	}
	return s.accrue(ctx, tx, now)
}

// completeActive finishes a Pomodoro whose time is up and turns its plot into a mature plant.
func (s *Service) completeActive(ctx context.Context, tx *sql.Tx, a *activeRow, now time.Time) error {
	if err := a.P.Complete(now); err != nil {
		return err
	}
	if err := saveActive(ctx, tx, a); err != nil {
		return err
	}
	if err := addEvent(ctx, tx, a.ID, "complete", *a.P.EndedAt); err != nil {
		return err
	}
	if !a.PlotID.Valid {
		return nil
	}
	matured := *a.P.EndedAt
	var lifeS int64
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(life_s, 0) FROM plots WHERE id = ?`, a.PlotID.Int64).Scan(&lifeS); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `UPDATE plots SET state = 'mature', matured_at = ?, wilts_at = ?, collected_to = ?,
		version = version + 1 WHERE id = ?`,
		fmtTime(matured), fmtTime(matured.Add(time.Duration(lifeS)*time.Second)), fmtTime(matured), a.PlotID.Int64)
	return err
}
