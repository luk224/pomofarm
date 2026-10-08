package api

import (
	"strings"
	"testing"
	"time"

	"github.com/luk224/pomofarm/backend/internal/service"
)

// Phase 4 QA: daylight-saving boundaries, month edges and a large history.

// Europe/Madrid leaves summer time on Sunday 25 October 2026 (03:00 → 02:00): the day has 25 hours and one hour happens twice.
func TestDaylightSavingDoesNotDuplicateOrSkipADay(t *testing.T) {
	e := newEnv(t, 1)
	for _, end := range []string{
		"2026-10-24T08:00:00Z", // Saturday 24 Oct, 10:00 CEST
		"2026-10-25T00:30:00Z", // Sunday, 02:30 CEST (first time)
		"2026-10-25T01:30:00Z", // Sunday, 02:30 CET (the repeated hour)
		"2026-10-26T09:00:00Z", // Monday 26 Oct, 10:00 CET
	} {
		addPomodoro(t, e, end, 1500, "x", "completed", 0)
	}
	_, b := getBook(t, e, "?month=2026-10&tz=Europe/Madrid")
	if b.Pomodoros != 4 || len(b.Days) != 3 {
		t.Fatalf("%d Pomodoros on %d days, want 4 on 3: %+v", b.Pomodoros, len(b.Days), b.Days)
	}
	for _, d := range b.Days {
		if d.Date == "2026-10-25" && d.Pomodoros != 2 {
			t.Fatalf("the long Sunday holds both 02:30s: %+v", d)
		}
	}
	if b.Weekdays[5].Pomodoros != 1 || b.Weekdays[6].Pomodoros != 2 || b.Weekdays[0].Pomodoros != 1 {
		t.Fatalf("Saturday 1, Sunday 2, Monday 1: %+v", b.Weekdays)
	}
	e.clock.T = time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	_, s := getStats(t, e, "?tz=Europe/Madrid")
	if s.BestStreakDays != 3 || s.CurrentStreak != 3 {
		t.Fatalf("Saturday, Sunday and Monday are three days in a row: best %d, current %d", s.BestStreakDays, s.CurrentStreak)
	}
	if last := s.Weeks[len(s.Weeks)-1]; last.Start != "2026-10-26" || last.Pomodoros != 1 {
		t.Fatalf("the week that starts on Monday 26 October: %+v", last)
	}
}

// Spring forward: Madrid skips 02:00–03:00 on Sunday 29 March 2026.
func TestSpringForwardKeepsWeeksAndStreaksIntact(t *testing.T) {
	e := newEnv(t, 1)
	for _, end := range []string{"2026-03-28T20:00:00Z", "2026-03-29T00:30:00Z", "2026-03-29T12:00:00Z", "2026-03-30T10:00:00Z"} {
		addPomodoro(t, e, end, 1500, "", "completed", 0)
	}
	e.clock.T = time.Date(2026, 3, 30, 15, 0, 0, 0, time.UTC)
	_, s := getStats(t, e, "?tz=Europe/Madrid")
	if s.BestStreakDays != 3 || s.CurrentStreak != 3 || s.Total != 4 {
		t.Fatalf("%+v", s)
	}
	var inWeeks int
	for _, w := range s.Weeks {
		inWeeks += w.Pomodoros
	}
	if inWeeks != 4 {
		t.Fatalf("every Pomodoro lands in exactly one week, got %d of 4", inWeeks)
	}
}

// A zone far from UTC moves a Pomodoro across the month boundary in both directions.
func TestMonthBoundariesInFarTimeZones(t *testing.T) {
	e := newEnv(t, 1)
	addPomodoro(t, e, "2026-09-30T20:00:00Z", 1500, "", "completed", 0) // 1 Oct in Auckland (UTC+13), 30 Sep in UTC and Honolulu
	addPomodoro(t, e, "2026-10-01T05:00:00Z", 1500, "", "completed", 0) // 30 Sep 19:00 in Honolulu (UTC-10)
	counts := map[string][2]int{}
	for _, tz := range []string{"UTC", "Pacific/Auckland", "Pacific/Honolulu"} {
		_, oct := getBook(t, e, "?month=2026-10&tz="+tz)
		_, sep := getBook(t, e, "?month=2026-09&tz="+tz)
		counts[tz] = [2]int{oct.Pomodoros, sep.Pomodoros}
	}
	want := map[string][2]int{"UTC": {1, 1}, "Pacific/Auckland": {2, 0}, "Pacific/Honolulu": {0, 2}}
	for tz, w := range want {
		if counts[tz] != w {
			t.Errorf("%s: (Oct, Sep) = %v, want %v", tz, counts[tz], w)
		}
	}
}

// Two years of intensive use: the statistics must stay fast and exact.
func TestALargeHistoryStaysFastAndExact(t *testing.T) {
	if testing.Short() {
		t.Skip("large history")
	}
	e := newEnv(t, 1)
	const days, perDay = 730, 12
	tx, err := e.db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	stmt, _ := tx.Prepare(`INSERT INTO pomodoros (player_id, plot_id, plant_type, tag_id, planned_s, started_at, paused_total_s, ended_at, reward_focus, status, strict) VALUES (1, NULL, 'daisy', NULL, 1500, ?, 0, ?, 1, 'completed', ?)`)
	end := e.clock.T.Add(-time.Hour)
	total := 0
	for d := 0; d < days; d++ {
		for k := 0; k < perDay; k++ {
			at := end.Add(-time.Duration(d)*24*time.Hour - time.Duration(k)*30*time.Minute)
			stmt.Exec(at.Add(-25*time.Minute).UTC().Format("2006-01-02T15:04:05.000000Z"), at.UTC().Format("2006-01-02T15:04:05.000000Z"), k%2)
			total++
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	for name, fn := range map[string]func() int{
		"book":  func() int { _, b := getBook(t, e, "?tz=Europe/Madrid"); return b.Pomodoros },
		"stats": func() int { _, s := getStats(t, e, "?tz=Europe/Madrid"); return s.Total },
		"csv":   func() int { recs, _ := csvOf(t, e, "?tz=Europe/Madrid"); return len(recs) - 1 },
	} {
		start := time.Now()
		n := fn()
		took := time.Since(start)
		t.Logf("%s over %d Pomodoros: %v (%d items)", name, total, took, n)
		if took > 3*time.Second {
			t.Errorf("%s took %v for %d Pomodoros", name, took, total)
		}
		if name != "book" && n != total {
			t.Errorf("%s counted %d of %d", name, n, total)
		}
	}
	_, s := getStats(t, e, "?tz=UTC")
	if s.BestStreakDays != days || s.StrictTotal != total/2 {
		t.Fatalf("a Pomodoro every day for %d days: best streak %d, strict %d of %d", days, s.BestStreakDays, s.StrictTotal, total)
	}
}

// Tags the player might type: nothing in the book or the CSV may break because of them.
func TestUnusualTagsSurviveTheBookAndTheCSV(t *testing.T) {
	e := newEnv(t, 1)
	tags := []string{"日本語", "emoji 🍅", "a,b;c\"d'e", "line\nbreak", "<script>alert(1)</script>", strings.Repeat("x", 60), "  spaced  ", "=SUM(A1)", "@cmd", "-1", "\ttab"}
	for _, tag := range tags {
		e.expect(201, "POST", "/api/pomodoros", service.PlantRequest{PlotID: 1, PlantType: "daisy", Tag: tag})
		e.clock.Advance(11 * time.Minute)
		e.expect(200, "POST", "/api/plots/1/harvest", nil)
		e.expect(200, "POST", "/api/plots/1/clear", map[string]bool{"confirm": true})
	}
	_, b := getBook(t, e, "?tz=UTC")
	if b.Pomodoros != len(tags) || len(b.Tags) != len(tags) {
		t.Fatalf("%d Pomodoros, %d tags (want %d each): %+v", b.Pomodoros, len(b.Tags), len(tags), b.Tags)
	}
	recs, _ := csvOf(t, e, "?tz=UTC")
	if len(recs) != len(tags)+1 {
		t.Fatalf("%d CSV rows", len(recs)-1)
	}
	for _, r := range recs[1:] {
		if first := r[3]; first != "" && strings.ContainsRune("=+-@\t\r", rune(first[0])) {
			t.Errorf("tag %q would run as a formula in a spreadsheet", first)
		}
		if len(r) != 11 {
			t.Errorf("row has %d columns: %v", len(r), r)
		}
	}
}
