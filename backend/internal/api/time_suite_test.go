package api

import (
	"fmt"
	"math/rand"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/luk224/pomofarm/backend/internal/game"
	"github.com/luk224/pomofarm/backend/internal/service"
	"github.com/luk224/pomofarm/backend/internal/store"
)

// The time suite (P5-03). GDD §8 and §6.1 make time the one thing the game must never get wrong: the server's timestamps are
// the only truth, nothing is counted by incrementing, and the player's clock never matters. docs/qa/tiempo.md lists which test
// proves which rule; this file adds the cases the earlier phases did not cover.

// A "restart" is closing the database and opening the same file with a brand-new service: nothing may live only in memory.
// unlockAllSeeds lets a test plant any crop without earning the 💧 for it.
func unlockAllSeeds(t *testing.T, e *env) {
	t.Helper()
	dbExec(t, e, `INSERT OR IGNORE INTO unlocks (player_id,kind,key,at) VALUES (1,'seed','tomato','t'),(1,'seed','sunflower','t'),(1,'seed','apple','t'),(1,'seed','oak','t')`)
}

func TestTheServerRestartingMidPomodoroLosesNothing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "restart.db")
	clock := &game.FakeClock{T: time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)}
	boot := func() *env {
		db, err := store.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { db.Close() })
		db.Exec(`INSERT OR IGNORE INTO players (id,name,created_at,last_seen_at) VALUES (1,'luk','t','t')`)
		db.Exec(`INSERT OR IGNORE INTO plots (player_id,x,y) VALUES (1,1,1)`)
		db.Exec(`INSERT OR IGNORE INTO unlocks (player_id,kind,key,at) VALUES (1,'seed','tomato','t')`)
		return &env{t: t, app: New(db, clock), db: db, clock: clock}
	}
	a := boot()
	a.expect(201, "POST", "/api/pomodoros", service.PlantRequest{PlotID: 1, PlantType: "tomato"}) // 25 min
	clock.Advance(10 * time.Minute)
	a.expect(200, "POST", "/api/pomodoros/active/pause", nil)
	clock.Advance(40 * time.Minute) // paused for 40 minutes
	a.db.Close()                    // the server goes down

	clock.Advance(3 * time.Hour) // ...and stays down for three hours, paused the whole time
	b := boot()
	st := b.state()
	if st.Pomodoro == nil || st.Pomodoro.Status != "paused" || st.Pomodoro.RemainingMs != 15*60*1000 {
		t.Fatalf("after the restart the paused Pomodoro must have exactly 15:00 left, got %+v", st.Pomodoro)
	}
	b.expect(200, "POST", "/api/pomodoros/active/resume", nil)
	clock.Advance(14*time.Minute + 59*time.Second)
	if b.state().Pomodoro == nil {
		t.Fatal("one second early it must still be running")
	}
	b.db.Close()
	clock.Advance(2 * time.Second) // it finishes while the server is down again
	c := boot()
	if st := c.state(); st.Pomodoro != nil || st.Plots[0].State != "mature" {
		t.Fatalf("a Pomodoro that ended while the server was down must be mature on the next start: %+v", st.Plots[0])
	}
}

func TestAPauseOfThirtyDaysChangesNothing(t *testing.T) {
	e := newEnv(t, 1)
	unlockAllSeeds(t, e)
	e.expect(201, "POST", "/api/pomodoros", service.PlantRequest{PlotID: 1, PlantType: "apple"}) // 45 min
	e.clock.Advance(20 * time.Minute)
	e.expect(200, "POST", "/api/pomodoros/active/pause", nil)
	e.clock.Advance(30 * 24 * time.Hour)
	if p := e.state().Pomodoro; p == nil || p.RemainingMs != 25*60*1000 || p.Status != "paused" {
		t.Fatalf("a month paused: %+v", p)
	}
	e.expect(200, "POST", "/api/pomodoros/active/resume", nil)
	e.clock.Advance(24*time.Minute + 59*time.Second)
	if e.state().Pomodoro == nil {
		t.Fatal("not finished one second early")
	}
	e.clock.Advance(2 * time.Second)
	if st := e.state(); st.Pomodoro != nil || st.Plots[0].State != "mature" {
		t.Fatal("finished exactly when its remaining time ran out")
	}
}

// The player's computer clock is not the game's clock. The server's is: jumps of it (NTP, a hand-set clock) must not break
// the running Pomodoro or invent coins.
func TestTheServerClockJumpingForwardAndBackMidPomodoro(t *testing.T) {
	e := newEnv(t, 1)
	e.expect(201, "POST", "/api/pomodoros", service.PlantRequest{PlotID: 1, PlantType: "daisy"}) // 10 min
	e.clock.Advance(4 * time.Minute)
	e.clock.Advance(-3 * time.Hour) // clock set back
	if p := e.state().Pomodoro; p == nil || p.RemainingMs > 10*60*1000 || p.RemainingMs < 0 {
		t.Fatalf("remaining must stay within [0, planned] when the clock is behind the start: %+v", p)
	}
	e.clock.Advance(3*time.Hour + 7*time.Minute) // …and corrected, 11 minutes after the start
	st := e.state()
	if st.Pomodoro != nil || st.Plots[0].State != "mature" {
		t.Fatalf("after the correction the Pomodoro is over: %+v", st.Pomodoro)
	}
	silo := st.Silo.ContentMilli
	e.clock.Advance(-5 * time.Hour) // set back again
	if again := e.state(); again.Silo.ContentMilli < silo || again.Plots[0].State != "mature" {
		t.Fatalf("going back never takes anything away: silo %d -> %d", silo, again.Silo.ContentMilli)
	}
	e.clock.Advance(5 * time.Hour) // forward again to the same instant
	if same := e.state(); same.Silo.ContentMilli != silo {
		t.Fatalf("back at the same instant nothing was counted twice: %d vs %d", silo, same.Silo.ContentMilli)
	}
}

func TestExtremeClocksNeitherPanicNorOverflow(t *testing.T) {
	for _, at := range []time.Time{
		time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2038, 1, 19, 3, 14, 8, 0, time.UTC), // the 32-bit limit
		time.Date(2100, 12, 31, 23, 59, 59, 999999999, time.UTC),
		time.Date(2262, 4, 11, 23, 47, 16, 0, time.UTC), // near the 64-bit nanosecond limit
	} {
		e := newEnv(t, 2)
		e.clock.T = at
		e.expect(201, "POST", "/api/pomodoros", service.PlantRequest{PlotID: 1, PlantType: "daisy"})
		e.clock.Advance(11 * time.Minute)
		st := e.state()
		if st.Pomodoro != nil || st.Plots[0].State != "mature" {
			t.Fatalf("at %v the Pomodoro did not complete: %+v", at, st.Plots[0])
		}
		e.clock.Advance(100 * 24 * time.Hour)
		if st := e.state(); st.Silo.ContentMilli < 0 || st.Silo.ContentMilli > st.Silo.CapacityMilli+2 {
			t.Fatalf("at %v the Silo is %d (capacity %d)", at, st.Silo.ContentMilli, st.Silo.CapacityMilli)
		}
	}
}

// The server's own time zone must not matter: nothing may use the local zone.
func TestTheServersTimeZoneNeverMatters(t *testing.T) {
	cycle := func() (int64, string) {
		e := newEnv(t, 1)
		unlockAllSeeds(t, e)
		e.expect(201, "POST", "/api/pomodoros", service.PlantRequest{PlotID: 1, PlantType: "tomato"})
		e.clock.Advance(26 * time.Minute)
		e.clock.Advance(7*time.Hour + 13*time.Minute)
		st := e.state()
		wilts := ""
		if st.Plots[0].WiltsAt != nil {
			wilts = *st.Plots[0].WiltsAt
		}
		return st.Silo.ContentMilli, wilts
	}
	saved := time.Local
	defer func() { time.Local = saved }()
	time.Local = time.UTC
	baseSilo, baseWilts := cycle()
	for _, z := range []*time.Location{time.FixedZone("Tokyo", 9*3600), time.FixedZone("LosAngeles", -8*3600), time.FixedZone("Kathmandu", 5*3600+45*60), time.FixedZone("Chatham", 12*3600+45*60)} {
		time.Local = z
		silo, wilts := cycle()
		if silo != baseSilo || wilts != baseWilts {
			t.Fatalf("with the server in %s: silo %d (want %d), wilts %s (want %s)", z, silo, baseSilo, wilts, baseWilts)
		}
	}
}

// Every instant the API sends is RFC 3339 in UTC with at most nine fractional digits: any browser's Date can read it
// (the client trims it to milliseconds).
func TestEveryTimestampTheAPISendsIsUTCRFC3339(t *testing.T) {
	e := newEnv(t, 2)
	e.expect(201, "POST", "/api/pomodoros", service.PlantRequest{PlotID: 1, PlantType: "daisy", Tag: "x"})
	e.clock.Advance(11 * time.Minute)
	e.expect(200, "POST", "/api/plots/1/harvest", nil)
	_, raw := e.do("GET", "/api/state", nil)
	re := regexp.MustCompile(`"(server_time|matured_at|wilts_at|started_at|paused_at|ends_at)":"([^"]+)"`)
	found := re.FindAllStringSubmatch(string(raw), -1)
	if len(found) < 4 {
		t.Fatalf("expected several timestamps in the state, found %d", len(found))
	}
	ok := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d{1,9})?Z$`)
	for _, m := range found {
		if !ok.MatchString(m[2]) {
			t.Errorf("%s = %q is not RFC 3339 UTC", m[1], m[2])
		}
		if _, err := time.Parse(time.RFC3339Nano, m[2]); err != nil {
			t.Errorf("%s = %q does not parse: %v", m[1], m[2], err)
		}
	}
}

// Asking for the state, at any rhythm, never changes where the game ends up: settling often equals settling once. This is
// the lazy-settlement promise (no cron job), proved over whole random games and not only over the Silo arithmetic.
// 💧, plants and every decision are identical; 🪙 may differ only by the rounding of each settled piece (a thousandth of a coin
// here and there, in either direction), never by a piece counted twice or lost.
func TestPollingNeverChangesWhereTheGameEndsUp(t *testing.T) {
	if testing.Short() {
		t.Skip("whole random games")
	}
	type outcome struct {
		coins, silo int64
		rest        string // everything that is not 🪙: 💧, plants, harvests
	}
	crops := []string{"daisy", "tomato", "sunflower", "apple", "oak"}
	play := func(seed int64, pollEvery time.Duration) outcome {
		rnd := rand.New(rand.NewSource(seed))
		e := newFreshEnv(t)
		giveFocus(t, e, 3000)
		for i := 0; i < 7; i++ {
			e.expect(201, "POST", "/api/plots", nil)
		}
		unlockAllSeeds(t, e)
		for step := 0; step < 90; step++ {
			switch op := rnd.Intn(10); {
			case op < 4:
				plots := e.state().Plots
				e.do("POST", "/api/pomodoros", service.PlantRequest{PlotID: plots[rnd.Intn(len(plots))].ID, PlantType: crops[rnd.Intn(5)]})
			case op < 6:
				plots := e.state().Plots
				e.do("POST", fmt.Sprintf("/api/plots/%d/harvest", plots[rnd.Intn(len(plots))].ID), nil)
			case op < 7:
				plots := e.state().Plots
				e.do("POST", fmt.Sprintf("/api/plots/%d/clear", plots[rnd.Intn(len(plots))].ID), map[string]bool{"confirm": true})
			case op < 8:
				e.do("POST", "/api/silo/collect", nil)
			case op < 9:
				e.do("POST", "/api/pomodoros/active/pause", nil)
			default:
				e.do("POST", "/api/pomodoros/active/resume", nil)
			}
			gap := time.Duration(rnd.Intn(4*60)+1) * time.Minute
			if pollEvery > 0 {
				for left := gap; left > 0; left -= pollEvery { // ask for the state many times inside the gap
					step := pollEvery
					if left < step {
						step = left
					}
					e.clock.Advance(step)
					e.state()
				}
			} else {
				e.clock.Advance(gap)
			}
		}
		e.clock.Advance(3 * 24 * time.Hour)
		st := e.state()
		var plots []string
		for _, p := range st.Plots {
			plots = append(plots, fmt.Sprintf("%d:%s:%v:%v", p.ID, p.State, p.PlantType != nil, p.Harvested))
		}
		return outcome{st.Player.CoinsMilli, st.Silo.ContentMilli, fmt.Sprintf("focus=%d life=%d %s", st.Player.FocusPoints, st.Player.LifetimeFocus, strings.Join(plots, ","))}
	}
	var worst int64
	for seed := int64(1); seed <= 5; seed++ {
		quiet := play(seed, 0)
		for _, every := range []time.Duration{5 * time.Minute, 37 * time.Minute} {
			got := play(seed, every)
			if got.rest != quiet.rest {
				t.Fatalf("seed %d: polling every %v changed 💧 or the plants\n  quiet:  %s\n  polled: %s", seed, every, quiet.rest, got.rest)
			}
			for name, d := range map[string]int64{"coins": got.coins - quiet.coins, "silo": got.silo - quiet.silo} {
				if d < 0 {
					d = -d
				}
				if d > worst {
					worst = d
				}
				if d > 10 {
					t.Fatalf("seed %d: polling every %v moved the %s by %d thousandths of a coin (quiet %+v, polled %+v)", seed, every, name, d, quiet, got)
				}
			}
		}
	}
	t.Logf("across 5 games × 2 polling rhythms the 🪙 never differed by more than %d thousandths of a coin", worst)
}

func TestAskingForTheStateManyTimesAtOneInstantChangesNothing(t *testing.T) {
	e := newEnv(t, 3)
	e.expect(201, "POST", "/api/pomodoros", service.PlantRequest{PlotID: 1, PlantType: "daisy"})
	e.clock.Advance(11 * time.Minute)
	e.clock.Advance(5 * time.Hour)
	first := e.state()
	for i := 0; i < 50; i++ {
		if got := e.state(); got.Silo.ContentMilli != first.Silo.ContentMilli || got.Player.CoinsMilli != first.Player.CoinsMilli || got.Player.FocusPoints != first.Player.FocusPoints {
			t.Fatalf("state request %d changed the game", i)
		}
	}
}
