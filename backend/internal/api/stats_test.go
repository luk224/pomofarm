package api

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/luk224/pomofarm/backend/internal/service"
)

func getStats(t *testing.T, e *env, query string) (int, service.Stats) {
	t.Helper()
	code, body := e.do("GET", "/api/stats"+query, nil)
	var s service.Stats
	if code == 200 {
		if err := json.Unmarshal(body, &s); err != nil {
			t.Fatalf("stats json: %v\n%s", err, body)
		}
	}
	return code, s
}

// runPomodoro plants a daisy (10 min) in plot 1, pauses once for each duration in `pauses`, lets it finish, then harvests
// and clears the plot so the next one can be planted.
func runPomodoro(t *testing.T, e *env, pauses ...time.Duration) {
	t.Helper()
	e.expect(201, "POST", "/api/pomodoros", service.PlantRequest{PlotID: 1, PlantType: "daisy"})
	for _, p := range pauses {
		e.clock.Advance(time.Minute)
		e.expect(200, "POST", "/api/pomodoros/active/pause", nil)
		e.clock.Advance(p)
		e.expect(200, "POST", "/api/pomodoros/active/resume", nil)
	}
	e.clock.Advance(11 * time.Minute)
	e.expect(200, "POST", "/api/plots/1/harvest", nil)
	e.expect(200, "POST", "/api/plots/1/clear", map[string]bool{"confirm": true})
}

func TestStatsStartEmpty(t *testing.T) {
	e := newEnv(t, 1)
	code, s := getStats(t, e, "")
	if code != 200 || s.Total != 0 || s.BestStreakDays != 0 || s.CurrentStreak != 0 || s.Clean != 0 || s.StrictTotal != 0 || s.StrictOn {
		t.Fatalf("%d %+v", code, s)
	}
	if len(s.Weeks) != 12 {
		t.Fatalf("%d weeks, want 12 (zeros included)", len(s.Weeks))
	}
	for i, w := range s.Weeks {
		if w.Pomodoros != 0 || (i > 0 && w.Start <= s.Weeks[i-1].Start) {
			t.Fatalf("weeks must be zeros in ascending order: %+v", s.Weeks)
		}
	}
}

func TestStatsWeeksAndTimeZone(t *testing.T) {
	e := newEnv(t, 1) // the fake clock starts on Sunday 2026-03-01 09:00 UTC
	// Sunday 1 March 23:30 UTC is Monday 2 March in Madrid (UTC+1 in March)
	addPomodoro(t, e, "2026-02-28T10:00:00Z", 1500, "", "completed", 0) // Saturday: week of 23 Feb
	addPomodoro(t, e, "2026-03-01T08:00:00Z", 1500, "", "completed", 0) // Sunday morning: still the week of 23 Feb
	addPomodoro(t, e, "2026-03-01T23:30:00Z", 1500, "", "completed", 0) // Sunday night UTC / Monday in Madrid
	addPomodoro(t, e, "2026-02-18T10:00:00Z", 1500, "", "cancelled", 0) // cancelled: never counts
	_, utc := getStats(t, e, "?tz=UTC")
	_, mad := getStats(t, e, "?tz=Europe/Madrid")
	last := func(s service.Stats) service.StatsWeek { return s.Weeks[len(s.Weeks)-1] }
	if last(utc).Start != "2026-02-23" || last(utc).Pomodoros != 3 || utc.ThisWeek != 3 {
		t.Fatalf("UTC current week (the clock reads Sunday 1 March): %+v this=%d", last(utc), utc.ThisWeek)
	}
	if mad.Total != 3 || utc.Total != 3 {
		t.Fatalf("totals %d %d", utc.Total, mad.Total)
	}
	// Madrid: the 23:30 UTC one belongs to the NEXT week (starting Monday 2 March), which is after "today" (Sunday, 10:00 local),
	// so the last listed week is still 23 Feb with 2 and the third Pomodoro is beyond the list: only Total shows it.
	if last(mad).Start != "2026-02-23" || last(mad).Pomodoros != 2 {
		t.Fatalf("Madrid current week: %+v", last(mad))
	}
}

func TestStatsStreaks(t *testing.T) {
	e := newEnv(t, 1) // today is 2026-03-01
	for _, d := range []string{"2026-02-10", "2026-02-11", "2026-02-12", "2026-02-13", "2026-02-27", "2026-02-28", "2026-03-01"} {
		addPomodoro(t, e, d+"T10:00:00Z", 1500, "", "completed", 0)
	}
	addPomodoro(t, e, "2026-02-12T11:00:00Z", 1500, "", "completed", 0) // two on one day are one day
	_, s := getStats(t, e, "?tz=UTC")
	if s.BestStreakDays != 4 || s.BestStreakEnd != "2026-02-13" || s.CurrentStreak != 3 {
		t.Fatalf("best %d ending %s, current %d", s.BestStreakDays, s.BestStreakEnd, s.CurrentStreak)
	}
	// a day with nothing does not take the best streak away
	e.clock.Advance(5 * 24 * time.Hour)
	_, s = getStats(t, e, "?tz=UTC")
	if s.BestStreakDays != 4 || s.CurrentStreak != 0 {
		t.Fatalf("after five idle days: best %d, current %d", s.BestStreakDays, s.CurrentStreak)
	}
}

func TestStrictModeCountsCleanPomodoros(t *testing.T) {
	e := newEnv(t, 1)
	e.expect(200, "POST", "/api/settings", map[string]string{"key": "strict_mode", "value": "1"})
	if _, s := getStats(t, e, ""); !s.StrictOn {
		t.Fatal("strict mode should read as on")
	}
	runPomodoro(t, e)                                        // 0 pauses: clean
	runPomodoro(t, e, 3*time.Minute, 4*time.Minute)          // 2 pauses, 7 min: clean
	runPomodoro(t, e, time.Minute, time.Minute, time.Minute) // 3 pauses: not clean
	runPomodoro(t, e, 11*time.Minute)                        // one pause of 11 min: not clean
	runPomodoro(t, e, 5*time.Minute, 5*time.Minute)          // 2 pauses, exactly 10 min: clean (limits are inclusive)
	_, s := getStats(t, e, "")
	if s.Total != 5 || s.StrictTotal != 5 || s.Clean != 3 {
		t.Fatalf("total %d, strict %d, clean %d (want 5, 5, 3)", s.Total, s.StrictTotal, s.Clean)
	}
}

func TestOnlyStrictPomodorosCanBeClean(t *testing.T) {
	e := newEnv(t, 1)
	runPomodoro(t, e) // strict mode off: perfectly uninterrupted, but not part of the challenge
	e.expect(200, "POST", "/api/settings", map[string]string{"key": "strict_mode", "value": "1"})
	runPomodoro(t, e)
	e.expect(200, "POST", "/api/settings", map[string]string{"key": "strict_mode", "value": "0"})
	runPomodoro(t, e)
	_, s := getStats(t, e, "")
	if s.Total != 3 || s.StrictTotal != 1 || s.Clean != 1 || s.StrictOn {
		t.Fatalf("%+v", s)
	}
}

func TestChangingStrictModeMidSessionDoesNotRewriteIt(t *testing.T) {
	e := newEnv(t, 1)
	e.expect(200, "POST", "/api/settings", map[string]string{"key": "strict_mode", "value": "1"})
	e.expect(201, "POST", "/api/pomodoros", service.PlantRequest{PlotID: 1, PlantType: "daisy"})
	e.expect(200, "POST", "/api/settings", map[string]string{"key": "strict_mode", "value": "0"}) // turned off while it runs
	e.clock.Advance(11 * time.Minute)
	if _, s := getStats(t, e, ""); s.StrictTotal != 1 || s.Clean != 1 {
		t.Fatalf("a session keeps the mode it started with: %+v", s)
	}
}

func TestACancelledStrictPomodoroIsNotCounted(t *testing.T) {
	e := newEnv(t, 1)
	e.expect(200, "POST", "/api/settings", map[string]string{"key": "strict_mode", "value": "1"})
	e.expect(201, "POST", "/api/pomodoros", service.PlantRequest{PlotID: 1, PlantType: "daisy"})
	e.expect(200, "POST", "/api/pomodoros/active/cancel", nil)
	if _, s := getStats(t, e, ""); s.Total != 0 || s.StrictTotal != 0 || s.Clean != 0 {
		t.Fatalf("%+v", s)
	}
}

func TestStrictModeNeverBlocksPausing(t *testing.T) {
	e := newEnv(t, 1)
	e.expect(200, "POST", "/api/settings", map[string]string{"key": "strict_mode", "value": "1"})
	e.expect(201, "POST", "/api/pomodoros", service.PlantRequest{PlotID: 1, PlantType: "daisy"})
	for i := 0; i < 6; i++ { // way past the limit: the game still lets the player pause (forgiving model)
		e.expect(200, "POST", "/api/pomodoros/active/pause", nil)
		e.clock.Advance(20 * time.Minute)
		e.expect(200, "POST", "/api/pomodoros/active/resume", nil)
	}
	e.clock.Advance(11 * time.Minute)
	if st := e.state(); st.Pomodoro != nil || st.Plots[0].State != "mature" {
		t.Fatal("the session must complete and pay as usual")
	}
}

func TestStrictSettingValidationAndCSVColumns(t *testing.T) {
	e := newEnv(t, 1)
	for _, v := range []string{"2", "yes", "", "-1"} {
		e.expect(400, "POST", "/api/settings", map[string]string{"key": "strict_mode", "value": v})
	}
	e.expect(200, "POST", "/api/settings", map[string]string{"key": "strict_mode", "value": "1"})
	runPomodoro(t, e)
	runPomodoro(t, e, time.Minute, time.Minute, time.Minute)
	e.expect(200, "POST", "/api/settings", map[string]string{"key": "strict_mode", "value": "0"})
	runPomodoro(t, e)
	recs, _ := csvOf(t, e, "")
	if len(recs) != 4 {
		t.Fatalf("%d rows", len(recs)-1)
	}
	for i, want := range [][2]string{{"1", "1"}, {"1", "0"}, {"0", "0"}} {
		if recs[i+1][9] != want[0] || recs[i+1][10] != want[1] {
			t.Fatalf("row %d estricto/limpio = %v, want %v", i+1, recs[i+1][9:], want)
		}
	}
}

func TestStatsRejectsABadTimeZone(t *testing.T) {
	e := newEnv(t, 1)
	for _, q := range []string{"?tz=Mars/Base", "?tz=%00", "?tz=" + string(make([]byte, 0)) + "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"} {
		if code, _ := getStats(t, e, q); code != 400 {
			t.Errorf("%s answered %d", q, code)
		}
	}
}

func TestTheActivePomodoroReportsItsStrictProgress(t *testing.T) {
	e := newEnv(t, 1)
	e.expect(200, "POST", "/api/settings", map[string]string{"key": "strict_mode", "value": "1"})
	e.expect(201, "POST", "/api/pomodoros", service.PlantRequest{PlotID: 1, PlantType: "daisy"})
	if p := e.state().Pomodoro; !p.Strict || p.Pauses != 0 || p.PausedMs != 0 {
		t.Fatalf("%+v", p)
	}
	e.clock.Advance(time.Minute)
	e.expect(200, "POST", "/api/pomodoros/active/pause", nil)
	e.clock.Advance(90 * time.Second)
	if p := e.state().Pomodoro; p.Pauses != 1 || p.PausedMs != 90_000 {
		t.Fatalf("while paused the pause in progress counts: %+v", p)
	}
	e.expect(200, "POST", "/api/pomodoros/active/resume", nil)
	e.clock.Advance(time.Minute)
	e.expect(200, "POST", "/api/pomodoros/active/pause", nil)
	e.clock.Advance(30 * time.Second)
	if p := e.state().Pomodoro; p.Pauses != 2 || p.PausedMs != 120_000 {
		t.Fatalf("%+v", p)
	}
}

func TestAPomodoroStartedWithoutStrictModeIsNotStrict(t *testing.T) {
	e := newEnv(t, 1)
	e.expect(201, "POST", "/api/pomodoros", service.PlantRequest{PlotID: 1, PlantType: "daisy"})
	if p := e.state().Pomodoro; p.Strict {
		t.Fatalf("%+v", p)
	}
}
