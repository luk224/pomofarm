package service

import (
	"context"
	"database/sql"
	"sort"
	"time"

	"github.com/luk224/pomofarm/backend/internal/game"
)

// Personal numbers (GDD §4.8): Pomodoros per week, the best streak ever and "clean" Pomodoros. Nothing here is a goal or a
// penalty: it only reports. A streak that ended costs nothing, and the best one is never lost.

const statsWeeks = 12

type StatsWeek struct {
	Start     string `json:"start"` // the Monday, YYYY-MM-DD in the player's time zone
	Pomodoros int    `json:"pomodoros"`
	Seconds   int64  `json:"seconds"`
}

type Stats struct {
	TimeZone string `json:"time_zone"`
	// Weeks are the last 12 weeks, oldest first; weeks without Pomodoros are present with zeros.
	Weeks          []StatsWeek `json:"weeks"`
	ThisWeek       int         `json:"this_week"`
	Total          int         `json:"total"`
	BestStreakDays int         `json:"best_streak_days"`
	BestStreakEnd  string      `json:"best_streak_end"`
	CurrentStreak  int         `json:"current_streak_days"`
	// StrictTotal are completed Pomodoros started in strict mode; Clean are those that also kept to its limits.
	StrictTotal int  `json:"strict_total"`
	Clean       int  `json:"clean"`
	StrictOn    bool `json:"strict_on"`
}

func (s *Service) Stats(ctx context.Context, tz string) (Stats, error) {
	loc, err := location(tz)
	if err != nil {
		return Stats{}, err
	}
	var out Stats
	err = s.withTx(ctx, func(tx *sql.Tx, now time.Time) error {
		if err := s.settle(ctx, tx, now); err != nil {
			return err
		}
		rows, err := loadBookRows(ctx, tx, `'completed'`)
		if err != nil {
			return err
		}
		today := now.In(loc).Format("2006-01-02")
		thisWeek := game.WeekStart(today)
		byWeek := map[string]*StatsWeek{}
		var days []string
		out = Stats{TimeZone: loc.String(), Weeks: []StatsWeek{}}
		for _, r := range rows {
			d := r.end.In(loc).Format("2006-01-02")
			days = append(days, d)
			w := game.WeekStart(d)
			if byWeek[w] == nil {
				byWeek[w] = &StatsWeek{Start: w}
			}
			byWeek[w].Pomodoros++
			byWeek[w].Seconds += r.plannedS
			out.Total++
			if r.strict {
				out.StrictTotal++
				if game.IsClean(true, r.pauses, r.pausedS) {
					out.Clean++
				}
			}
		}
		start, _ := time.Parse("2006-01-02", thisWeek)
		for i := statsWeeks - 1; i >= 0; i-- {
			key := start.AddDate(0, 0, -7*i).Format("2006-01-02")
			if w := byWeek[key]; w != nil {
				out.Weeks = append(out.Weeks, *w)
			} else {
				out.Weeks = append(out.Weeks, StatsWeek{Start: key})
			}
		}
		sort.Slice(out.Weeks, func(i, j int) bool { return out.Weeks[i].Start < out.Weeks[j].Start })
		if w := byWeek[thisWeek]; w != nil {
			out.ThisWeek = w.Pomodoros
		}
		out.BestStreakDays, out.BestStreakEnd, out.CurrentStreak = game.Streaks(days, today)
		var v string
		_ = tx.QueryRowContext(ctx, `SELECT value FROM settings WHERE player_id = ? AND key = 'strict_mode'`, PlayerID).Scan(&v)
		out.StrictOn = v == "1"
		return nil
	})
	return out, err
}
