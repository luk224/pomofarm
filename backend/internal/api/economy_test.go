package api

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/luk224/pomofarm/backend/internal/service"
)

func milli(t *testing.T, e *env) (silo, coins int64) {
	t.Helper()
	st := e.state()
	return st.Silo.ContentMilli, st.Player.CoinsMilli
}

func near(t *testing.T, name string, got, want, tol int64) {
	t.Helper()
	if d := got - want; d < -tol || d > tol {
		t.Fatalf("%s = %d, want %d ±%d", name, got, want, tol)
	}
}

// plantAndMature leaves a mature, harvested daisy on plot 1 at the current fake time.
func plantAndMature(t *testing.T, e *env) {
	t.Helper()
	e.expect(201, "POST", "/api/pomodoros", service.PlantRequest{PlotID: 1, PlantType: "daisy"})
	e.clock.Advance(10 * time.Minute)
	e.expect(200, "POST", "/api/plots/1/harvest", nil)
}

// A daisy yields 20 🪙 over its 24 h life: 0.8333 🪙/h.
func TestMaturePlantProducesIntoTheSilo(t *testing.T) {
	e := newFreshEnv(t)
	plantAndMature(t, e)
	e.clock.Advance(5 * time.Hour)
	st := e.state()
	near(t, "silo after 5 h", st.Silo.ContentMilli, 4166, 1)
	near(t, "current rate 🪙/h ×1000", st.Silo.RateMilliPerHour, 833, 1)
	near(t, "capacity (12 h × peak)", st.Silo.CapacityMilli, 10000, 2)
	if st.Silo.CapacityHours != 12 || st.Player.CoinsMilli != 0 || st.Silo.Full {
		t.Fatalf("silo = %+v coins = %d", st.Silo, st.Player.CoinsMilli)
	}
}

func TestNothingIsProducedWhileGrowing(t *testing.T) {
	e := newFreshEnv(t)
	e.expect(201, "POST", "/api/pomodoros", service.PlantRequest{PlotID: 1, PlantType: "daisy"})
	e.clock.Advance(9 * time.Minute)
	if silo, _ := milli(t, e); silo != 0 {
		t.Fatalf("silo = %d while the plant is still growing", silo)
	}
}

func TestCollectMovesTheSiloToTheBalanceWithoutLosingFractions(t *testing.T) {
	e := newFreshEnv(t)
	plantAndMature(t, e)
	e.clock.Advance(5 * time.Hour)
	var out struct {
		Collected int64         `json:"collected_milli"`
		State     service.State `json:"state"`
	}
	json.Unmarshal(e.expect(200, "POST", "/api/silo/collect", nil), &out)
	near(t, "collected", out.Collected, 4166, 1)
	if out.State.Player.CoinsMilli != out.Collected || out.State.Silo.ContentMilli != 0 {
		t.Fatalf("after collect: coins %d, silo %d", out.State.Player.CoinsMilli, out.State.Silo.ContentMilli)
	}
	if code := errCode(e.expect(409, "POST", "/api/silo/collect", nil)); code != "silo_empty" {
		t.Fatalf("second collect: %s", code)
	}
	// the sub-thousandth remainder stayed in the Silo: 5 h + 1 h of production = exactly 5000 🪙/1000
	e.clock.Advance(1 * time.Hour)
	e.expect(200, "POST", "/api/silo/collect", nil)
	_, coins := milli(t, e)
	near(t, "total coins after 6 h in two collections", coins, 5000, 1)
}

func TestFullSiloLosesProductionUntilEmptied(t *testing.T) {
	e := newFreshEnv(t)
	plantAndMature(t, e)
	e.clock.Advance(30 * time.Hour) // the daisy made 20 🪙 in its 24 h of life; the 12 h Silo holds 10
	st := e.state()
	near(t, "silo", st.Silo.ContentMilli, 10000, 2)
	if !st.Silo.Full {
		t.Fatal("Silo should be full")
	}
	e.expect(200, "POST", "/api/silo/collect", nil)
	e.clock.Advance(40 * time.Hour) // the plant is dead: nothing more, ever
	silo, coins := milli(t, e)
	near(t, "coins", coins, 10000, 2)
	if silo != 0 {
		t.Fatalf("a wilted plant kept producing: silo %d", silo)
	}
}

func TestNinetyDaysAwayIsBounded(t *testing.T) {
	e := newFreshEnv(t)
	plantAndMature(t, e)
	e.clock.Advance(90 * 24 * time.Hour)
	st := e.state()
	if st.Silo.ContentMilli > 10002 || st.Silo.ContentMilli > 20000 {
		t.Fatalf("90 days away stored %d, more than the 12 h Silo (10000) allows", st.Silo.ContentMilli)
	}
	e.expect(200, "POST", "/api/silo/collect", nil)
	if _, coins := milli(t, e); coins > 20000 {
		t.Fatalf("collected %d: more than the plant's whole life can produce (20000)", coins)
	}
}

func TestClockGoingBackwardsNeitherProducesNorDoubleCounts(t *testing.T) {
	e := newFreshEnv(t)
	plantAndMature(t, e)
	e.clock.Advance(5 * time.Hour)
	base, _ := milli(t, e)
	e.clock.Advance(-2 * time.Hour)
	if silo, _ := milli(t, e); silo != base {
		t.Fatalf("clock back 2 h: silo %d, was %d", silo, base)
	}
	e.clock.Advance(2 * time.Hour) // back to the same instant: still nothing extra
	if silo, _ := milli(t, e); silo != base {
		t.Fatalf("returning to the same instant changed the silo: %d vs %d", silo, base)
	}
	e.clock.Advance(1 * time.Hour)
	silo, _ := milli(t, e)
	near(t, "one more hour", silo-base, 833, 2)
}

func TestClearingAPlantStopsItsProductionButKeepsWhatItMade(t *testing.T) {
	e := newFreshEnv(t)
	plantAndMature(t, e)
	e.clock.Advance(2 * time.Hour)
	e.expect(200, "POST", "/api/plots/1/clear", map[string]bool{"confirm": true})
	silo, _ := milli(t, e)
	near(t, "made before clearing", silo, 1666, 1)
	e.clock.Advance(10 * time.Hour)
	if after, _ := milli(t, e); after != silo {
		t.Fatalf("an empty plot produced: %d -> %d", silo, after)
	}
}

func TestSettlingOftenEqualsSettlingOnce(t *testing.T) {
	once, often := newFreshEnv(t), newFreshEnv(t)
	plantAndMature(t, once)
	plantAndMature(t, often)
	once.clock.Advance(20 * time.Hour)
	for i := 0; i < 80; i++ {
		often.clock.Advance(15 * time.Minute)
		often.state() // every request settles
	}
	a, _ := milli(t, once)
	b, _ := milli(t, often)
	near(t, "20 h in 80 requests vs 1", b, a, 1)
}

func TestConcurrentRequestsDoNotDoubleCount(t *testing.T) {
	e := newFreshEnv(t)
	plantAndMature(t, e)
	e.clock.Advance(5 * time.Hour)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); e.do("GET", "/api/state", nil) }()
	}
	wg.Wait()
	silo, _ := milli(t, e)
	near(t, "silo after 8 concurrent requests", silo, 4166, 1)
}

func TestTheDogCollectsByItselfAndAddsTwelveHours(t *testing.T) {
	e := newFreshEnv(t)
	dbExec(t, e, `INSERT INTO structures (player_id, kind, x, y) VALUES (1, 'dog', 0, 0)`)
	plantAndMature(t, e)
	e.clock.Advance(5 * time.Hour)
	st := e.state()
	near(t, "coins collected by the Dog", st.Player.CoinsMilli, 4166, 1)
	if st.Silo.ContentMilli != 0 || st.Silo.CapacityHours != 24 {
		t.Fatalf("silo %+v, want empty and 24 h (12 + the Dog's 12)", st.Silo)
	}
}

// GDD §6.2 and §8: the worked example through the whole stack, with both Silo capacities.
// Three plants matured 30 h ago and are collected now: Tomatoes wilt in 6 h, the Sunflower in 20 h, the Apple lives on.
func TestGDDExampleWithTheTwoSiloCaps(t *testing.T) {
	for _, c := range []struct {
		name     string
		level    int
		lo, hi   float64 // 🪙
		capHours int
	}{
		{"Silo de 24 h (nivel 1): no trunca, 212,9", 1, 212.8, 213.0, 24},
		{"Silo de 12 h (nivel 0): se llena, 127,0", 0, 126.9, 127.2, 12},
	} {
		t.Run(c.name, func(t *testing.T) {
			e := newFreshEnv(t)
			// The GDD example assumes M = 1: put the plants far apart so no synergy applies (they have their own tests).
			dbExec(t, e, `INSERT INTO plots (player_id, x, y) VALUES (1, 3, 3), (1, 3, 0)`)
			now := e.clock.T
			t0 := now.Format(time.RFC3339Nano)
			ago := now.Add(-30 * time.Hour).Format(time.RFC3339Nano)
			wilt := func(h int) string { return now.Add(time.Duration(h) * time.Hour).Format(time.RFC3339Nano) }
			for _, p := range []struct {
				id                        int
				plant                     string
				growS, lifeH, wiltInHours int
			}{
				{1, "tomato", 25 * 60, 36, 6}, {2, "sunflower", 35 * 60, 54, 20}, {3, "apple", 45 * 60, 72, 42},
			} {
				dbExec(t, e, `UPDATE plots SET state='mature', plant_type=?, grow_s=?, life_s=?, matured_at=?, wilts_at=?, collected_to=?, harvested=1 WHERE id=?`,
					p.plant, p.growS, p.lifeH*3600, ago, wilt(p.wiltInHours), t0, p.id)
			}
			dbExec(t, e, `UPDATE players SET silo_level=?`, c.level)
			e.clock.Advance(30 * time.Hour)
			st := e.state()
			got := float64(st.Silo.ContentMilli) / 1000
			if got < c.lo || got > c.hi {
				t.Fatalf("Silo holds %.2f 🪙, want %.1f–%.1f", got, c.lo, c.hi)
			}
			if st.Silo.CapacityHours != c.capHours {
				t.Fatalf("capacity %d h", st.Silo.CapacityHours)
			}
		})
	}
}

// ---- P2-02: life span and withering ----

func plot1(t *testing.T, e *env) service.PlotState {
	t.Helper()
	return e.state().Plots[0]
}

// A daisy matures 10 min after planting and lives 24 h: it wilts exactly 24 h 10 min after planting.
func TestPlantWiltsExactlyWhenItsLifeEnds(t *testing.T) {
	e := newFreshEnv(t)
	e.expect(201, "POST", "/api/pomodoros", service.PlantRequest{PlotID: 1, PlantType: "daisy"})
	e.clock.Advance(10 * time.Minute)
	e.state()
	e.clock.Advance(24*time.Hour - time.Second)
	if st := plot1(t, e).State; st != "mature" {
		t.Fatalf("1 s before the end of its life: %s", st)
	}
	e.clock.Advance(time.Second)
	if st := plot1(t, e).State; st != "withered" {
		t.Fatalf("at the end of its life: %s", st)
	}
}

func TestWitheredPlantStopsProducingAndTheSiloKeepsWhatItMade(t *testing.T) {
	e := newFreshEnv(t)
	plantAndMature(t, e)
	e.clock.Advance(48 * time.Hour)
	st := e.state()
	if st.Plots[0].State != "withered" || st.Silo.RateMilliPerHour != 0 {
		t.Fatalf("plot %s, rate %d", st.Plots[0].State, st.Silo.RateMilliPerHour)
	}
	near(t, "Silo holds its 12 h capacity (the plant made 20, the Silo holds 10)", st.Silo.ContentMilli, 10000, 2)
	e.expect(200, "POST", "/api/silo/collect", nil)
	e.clock.Advance(100 * time.Hour)
	if silo, _ := milli(t, e); silo != 0 {
		t.Fatalf("a withered plant kept producing: %d", silo)
	}
}

func TestWitheredPlantKeepsItsRewardUntilHarvested(t *testing.T) {
	e := newFreshEnv(t)
	e.expect(201, "POST", "/api/pomodoros", service.PlantRequest{PlotID: 1, PlantType: "daisy"})
	e.clock.Advance(30 * time.Hour) // finished, never harvested, and wilted while the player was away
	st := e.state()
	if st.Plots[0].State != "withered" || st.Plots[0].Harvested {
		t.Fatalf("plot = %+v", st.Plots[0])
	}
	var out struct {
		Reward int `json:"reward_focus"`
	}
	json.Unmarshal(e.expect(200, "POST", "/api/plots/1/harvest", nil), &out)
	if out.Reward != 1 || e.state().Player.FocusPoints != 1 {
		t.Fatalf("reward %d, focus %d: a withered plant must still pay its 💧", out.Reward, e.state().Player.FocusPoints)
	}
	if code := errCode(e.expect(409, "POST", "/api/plots/1/harvest", nil)); code != "already_harvested" {
		t.Fatalf("second harvest: %s", code)
	}
}

func TestClearingAWitheredPlantIsFreeAndNeedsNoConfirmation(t *testing.T) {
	e := newFreshEnv(t)
	plantAndMature(t, e)
	e.clock.Advance(48 * time.Hour)
	e.expect(200, "POST", "/api/plots/1/clear", nil) // no {"confirm": true}
	if st := plot1(t, e); st.State != "empty" || st.PlantType != nil {
		t.Fatalf("after clear: %+v", st)
	}
}

func TestWitheredRewardIsProtectedFromClearAndReplant(t *testing.T) {
	e := newFreshEnv(t)
	e.expect(201, "POST", "/api/pomodoros", service.PlantRequest{PlotID: 1, PlantType: "daisy"})
	e.clock.Advance(30 * time.Hour)
	if code := errCode(e.expect(409, "POST", "/api/plots/1/clear", nil)); code != "harvest_first" {
		t.Fatalf("clear before harvest: %s", code)
	}
	if code := errCode(e.expect(409, "POST", "/api/pomodoros", service.PlantRequest{PlotID: 1, PlantType: "daisy"})); code != "harvest_first" {
		t.Fatalf("plant over an unharvested withered plant: %s", code)
	}
	e.expect(200, "POST", "/api/plots/1/harvest", nil)
	e.expect(201, "POST", "/api/pomodoros", service.PlantRequest{PlotID: 1, PlantType: "daisy"}) // now replanting is fine
}

func TestAMatureLivePlantStillNeedsConfirmationToClear(t *testing.T) {
	e := newFreshEnv(t)
	plantAndMature(t, e)
	e.clock.Advance(2 * time.Hour)
	if code := errCode(e.expect(409, "POST", "/api/plots/1/clear", nil)); code != "needs_confirmation" {
		t.Fatalf("code = %s", code)
	}
}

// The reward comes from what the plot remembers (type, duration), not from a history row that could be missing.
func TestHarvestDoesNotDependOnPomodoroHistory(t *testing.T) {
	e := newFreshEnv(t)
	e.expect(201, "POST", "/api/pomodoros", service.PlantRequest{PlotID: 1, PlantType: "daisy"})
	e.clock.Advance(10 * time.Minute)
	e.state() // the plant matures when a request settles the Pomodoro
	dbExec(t, e, `DELETE FROM pomodoro_events`)
	dbExec(t, e, `DELETE FROM pomodoros`)
	e.expect(200, "POST", "/api/plots/1/harvest", nil)
	if got := e.state().Player.FocusPoints; got != 1 {
		t.Fatalf("focus = %d, want 1", got)
	}
}
