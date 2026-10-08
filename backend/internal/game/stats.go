package game

import "time"

// Strict mode (GDD §3.3): at most two pauses and ten minutes of pause in total. It is a personal challenge with no economic
// effect, and it never stops the player from pausing: it only decides whether a finished Pomodoro was "clean".
const (
	StrictMaxPauses  = 2
	StrictMaxPausedS = 10 * 60
)

// IsClean reports whether a Pomodoro that was started in strict mode kept to the limits.
func IsClean(strict bool, pauses int, pausedS int64) bool {
	return strict && pauses <= StrictMaxPauses && pausedS <= StrictMaxPausedS
}

// Streaks finds, among the days (YYYY-MM-DD, any order, repeats allowed) on which at least one Pomodoro was completed,
// the longest run of consecutive days ever, the day it ended, and the run that is still going on `today`.
// A run that ended yesterday is still "current": today is not over yet. Breaking a streak costs nothing (GDD §4.8).
func Streaks(days []string, today string) (best int, bestEnd string, current int) {
	set := map[string]bool{}
	for _, d := range days {
		if _, err := time.Parse("2006-01-02", d); err == nil {
			set[d] = true
		}
	}
	for d := range set {
		t, _ := time.Parse("2006-01-02", d)
		if set[t.AddDate(0, 0, -1).Format("2006-01-02")] {
			continue // not the first day of a run
		}
		n := 0
		for cur := t; set[cur.Format("2006-01-02")]; cur = cur.AddDate(0, 0, 1) {
			n++
		}
		end := t.AddDate(0, 0, n-1).Format("2006-01-02")
		if n > best || (n == best && end > bestEnd) {
			best, bestEnd = n, end
		}
	}
	t, err := time.Parse("2006-01-02", today)
	if err != nil {
		return best, bestEnd, 0
	}
	if !set[today] {
		t = t.AddDate(0, 0, -1)
	}
	for set[t.Format("2006-01-02")] {
		current++
		t = t.AddDate(0, 0, -1)
	}
	return best, bestEnd, current
}

// WeekStart is the Monday (YYYY-MM-DD) of the week that contains `date`.
func WeekStart(date string) string {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return ""
	}
	back := (int(t.Weekday()) + 6) % 7
	return t.AddDate(0, 0, -back).Format("2006-01-02")
}
