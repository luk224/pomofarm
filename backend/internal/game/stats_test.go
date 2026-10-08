package game

import "testing"

func TestIsClean(t *testing.T) {
	cases := []struct {
		strict  bool
		pauses  int
		pausedS int64
		want    bool
	}{
		{true, 0, 0, true}, {true, 2, 600, true}, {true, 3, 0, false}, {true, 1, 601, false}, {true, 2, 599, true},
		{false, 0, 0, false}, {false, 1, 10, false}, // without strict mode nothing is "clean": it is a challenge you opt into
	}
	for _, c := range cases {
		if got := IsClean(c.strict, c.pauses, c.pausedS); got != c.want {
			t.Errorf("IsClean(%v, %d, %d) = %v, want %v", c.strict, c.pauses, c.pausedS, got, c.want)
		}
	}
}

func TestStreaks(t *testing.T) {
	cases := []struct {
		name    string
		days    []string
		today   string
		best    int
		bestEnd string
		current int
	}{
		{"nothing", nil, "2026-10-08", 0, "", 0},
		{"only today", []string{"2026-10-08"}, "2026-10-08", 1, "2026-10-08", 1},
		{"a run that ended yesterday is still current", []string{"2026-10-06", "2026-10-07"}, "2026-10-08", 2, "2026-10-07", 2},
		{"missing yesterday and today: no current streak, the best stays", []string{"2026-10-05", "2026-10-06", "2026-10-07"}, "2026-10-09", 3, "2026-10-07", 0},
		{"repeats and order do not matter", []string{"2026-10-07", "2026-10-05", "2026-10-06", "2026-10-06", "2026-10-05"}, "2026-10-07", 3, "2026-10-07", 3},
		{"the best run is the longest, not the latest", []string{"2026-09-01", "2026-09-02", "2026-09-03", "2026-10-07", "2026-10-08"}, "2026-10-08", 3, "2026-09-03", 2},
		{"a tie goes to the most recent run", []string{"2026-09-01", "2026-09-02", "2026-10-07", "2026-10-08"}, "2026-10-08", 2, "2026-10-08", 2},
		{"across a month and a year", []string{"2025-12-30", "2025-12-31", "2026-01-01", "2026-01-02"}, "2026-01-02", 4, "2026-01-02", 4},
		{"leap day", []string{"2028-02-28", "2028-02-29", "2028-03-01"}, "2028-03-01", 3, "2028-03-01", 3},
		{"garbage dates are ignored", []string{"x", "", "2026-13-40", "2026-10-08"}, "2026-10-08", 1, "2026-10-08", 1},
		{"bad today gives no current streak", []string{"2026-10-08"}, "nope", 1, "2026-10-08", 0},
	}
	for _, c := range cases {
		best, end, cur := Streaks(c.days, c.today)
		if best != c.best || end != c.bestEnd || cur != c.current {
			t.Errorf("%s: got (%d, %q, %d), want (%d, %q, %d)", c.name, best, end, cur, c.best, c.bestEnd, c.current)
		}
	}
}

func TestWeekStart(t *testing.T) {
	for in, want := range map[string]string{
		"2026-10-05": "2026-10-05", "2026-10-07": "2026-10-05", "2026-10-11": "2026-10-05", "2026-10-12": "2026-10-12",
		"2026-01-01": "2025-12-29", "2028-03-01": "2028-02-28", "bad": "",
	} {
		if got := WeekStart(in); got != want {
			t.Errorf("WeekStart(%s) = %q, want %q", in, got, want)
		}
	}
}
