package service

import (
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/luk224/pomofarm/backend/internal/game"
)

// The Harvest Book (GDD §4.8): focus time per month, by tag and by weekday, and a CSV of every Pomodoro. Days and
// weekdays are the player's own, so the browser sends its IANA time zone; storage stays in UTC.

type BookDay struct {
	Date      string `json:"date"` // YYYY-MM-DD in the player's time zone
	Pomodoros int    `json:"pomodoros"`
	Seconds   int64  `json:"seconds"`
}

type BookTag struct {
	Name      string `json:"name"` // empty for Pomodoros without a tag
	Pomodoros int    `json:"pomodoros"`
	Seconds   int64  `json:"seconds"`
}

type BookWeekday struct {
	Weekday   int   `json:"weekday"` // 0 = Monday … 6 = Sunday
	Pomodoros int   `json:"pomodoros"`
	Seconds   int64 `json:"seconds"`
}

type BookMonth struct {
	Month     string        `json:"month"`
	TimeZone  string        `json:"time_zone"`
	Pomodoros int           `json:"pomodoros"`
	Seconds   int64         `json:"seconds"`
	Days      []BookDay     `json:"days"`
	Tags      []BookTag     `json:"tags"`
	Weekdays  []BookWeekday `json:"weekdays"`
	// Months lists every month that has data, plus the current one, oldest first, for the month picker.
	Months []string `json:"months"`
}

type bookRow struct {
	id         int64
	start, end time.Time
	tag, plant string
	plannedS   int64
	pausedS    int64
	reward     int64
	status     string
	strict     bool
	pauses     int
}

// location validates an IANA time-zone name; empty means UTC.
func location(name string) (*time.Location, error) {
	if name == "" {
		return time.UTC, nil
	}
	if len(name) > 64 {
		return nil, ErrInvalid
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return nil, ErrInvalid
	}
	return loc, nil
}

func loadBookRows(ctx context.Context, tx *sql.Tx, statuses string) ([]bookRow, error) {
	rows, err := tx.QueryContext(ctx, `SELECT p.id, p.started_at, p.ended_at, COALESCE(t.name, ''), p.plant_type, p.planned_s, p.paused_total_s, p.reward_focus, p.status, p.strict,
		(SELECT COUNT(*) FROM pomodoro_events e WHERE e.pomodoro_id = p.id AND e.kind = 'pause')
		FROM pomodoros p LEFT JOIN tags t ON t.id = p.tag_id
		WHERE p.player_id = ? AND p.status IN (`+statuses+`) AND p.ended_at IS NOT NULL ORDER BY p.ended_at, p.id`, PlayerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []bookRow
	for rows.Next() {
		var r bookRow
		var start, end string
		if err := rows.Scan(&r.id, &start, &end, &r.tag, &r.plant, &r.plannedS, &r.pausedS, &r.reward, &r.status, &r.strict, &r.pauses); err != nil {
			return nil, err
		}
		r.start, r.end = parseTime(start), parseTime(end)
		out = append(out, r)
	}
	return out, rows.Err()
}

// Book is one month of the Harvest Book. `month` is YYYY-MM (empty = the current month in the player's zone).
// Only completed Pomodoros count; the time is the planned focus time (pauses are not focus).
func (s *Service) Book(ctx context.Context, month, tz string) (BookMonth, error) {
	loc, err := location(tz)
	if err != nil {
		return BookMonth{}, err
	}
	var out BookMonth
	err = s.withTx(ctx, func(tx *sql.Tx, now time.Time) error {
		if err := s.settle(ctx, tx, now); err != nil {
			return err
		}
		if month == "" {
			month = now.In(loc).Format("2006-01")
		}
		if _, err := time.ParseInLocation("2006-01", month, loc); err != nil {
			return ErrInvalid
		}
		rows, err := loadBookRows(ctx, tx, `'completed'`)
		if err != nil {
			return err
		}
		out = BookMonth{Month: month, TimeZone: loc.String(), Days: []BookDay{}, Tags: []BookTag{}, Weekdays: make([]BookWeekday, 7)}
		for i := range out.Weekdays {
			out.Weekdays[i].Weekday = i
		}
		have := map[string]bool{now.In(loc).Format("2006-01"): true}
		days := map[string]*BookDay{}
		tags := map[string]*BookTag{}
		for _, r := range rows {
			local := r.end.In(loc)
			have[local.Format("2006-01")] = true
			if local.Format("2006-01") != month {
				continue
			}
			out.Pomodoros++
			out.Seconds += r.plannedS
			d := local.Format("2006-01-02")
			if days[d] == nil {
				days[d] = &BookDay{Date: d}
			}
			days[d].Pomodoros++
			days[d].Seconds += r.plannedS
			if tags[r.tag] == nil {
				tags[r.tag] = &BookTag{Name: r.tag}
			}
			tags[r.tag].Pomodoros++
			tags[r.tag].Seconds += r.plannedS
			wd := (int(local.Weekday()) + 6) % 7 // Monday first
			out.Weekdays[wd].Pomodoros++
			out.Weekdays[wd].Seconds += r.plannedS
		}
		for _, d := range days {
			out.Days = append(out.Days, *d)
		}
		sort.Slice(out.Days, func(i, j int) bool { return out.Days[i].Date < out.Days[j].Date })
		for _, t := range tags {
			out.Tags = append(out.Tags, *t)
		}
		sort.Slice(out.Tags, func(i, j int) bool {
			if out.Tags[i].Seconds != out.Tags[j].Seconds {
				return out.Tags[i].Seconds > out.Tags[j].Seconds
			}
			return out.Tags[i].Name < out.Tags[j].Name
		})
		for m := range have {
			out.Months = append(out.Months, m)
		}
		sort.Strings(out.Months)
		return nil
	})
	return out, err
}

// csvSafe stops a spreadsheet from running a tag as a formula: a leading = + - @ (or tab / CR) gets an apostrophe.
func csvSafe(s string) string {
	if s != "" && strings.ContainsRune("=+-@\t\r", rune(s[0])) {
		return "'" + s
	}
	return s
}

// BookCSV is every finished Pomodoro (completed or cancelled), one row each, in the player's time zone. It starts with a
// UTF-8 byte-order mark so Excel reads accents correctly.
func (s *Service) BookCSV(ctx context.Context, tz string) ([]byte, error) {
	loc, err := location(tz)
	if err != nil {
		return nil, err
	}
	var rows []bookRow
	err = s.withTx(ctx, func(tx *sql.Tx, now time.Time) error {
		if err := s.settle(ctx, tx, now); err != nil {
			return err
		}
		rows, err = loadBookRows(ctx, tx, `'completed','cancelled'`)
		return err
	})
	if err != nil {
		return nil, err
	}
	var b strings.Builder
	b.WriteString("\uFEFF")
	w := csv.NewWriter(&b)
	_ = w.Write([]string{"id", "inicio", "fin", "etiqueta", "cultivo", "minutos", "pausa_minutos", "gotas", "estado", "estricto", "limpio"})
	for _, r := range rows {
		_ = w.Write([]string{
			fmt.Sprint(r.id), r.start.In(loc).Format("2006-01-02 15:04:05"), r.end.In(loc).Format("2006-01-02 15:04:05"),
			csvSafe(r.tag), r.plant, fmt.Sprintf("%.1f", float64(r.plannedS)/60), fmt.Sprintf("%.1f", float64(r.pausedS)/60), fmt.Sprint(r.reward), r.status, boolBit(r.strict), boolBit(r.status == "completed" && game.IsClean(r.strict, r.pauses, r.pausedS)),
		})
	}
	w.Flush()
	return []byte(b.String()), w.Error()
}

func boolBit(b bool) string {
	if b {
		return "1"
	}
	return "0"
}
