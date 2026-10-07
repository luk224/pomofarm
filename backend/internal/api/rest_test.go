package api

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/luk224/pomofarm/backend/internal/service"
)

// finish plants `crop` for `minutes`, lets it mature and harvests it `after` later; it returns the state.
func finishAndHarvest(t *testing.T, e *env, crop string, minutes int, after time.Duration) service.State {
	t.Helper()
	e.expect(201, "POST", "/api/pomodoros", service.PlantRequest{PlotID: 1, PlantType: crop, DurationMin: minutes})
	e.clock.Advance(time.Duration(e.state().Pomodoro.PlannedS)*time.Second + after)
	e.expect(200, "POST", "/api/plots/1/harvest", nil)
	return e.state()
}

func restMinutes(st service.State) float64 {
	if st.Rest == nil {
		return 0
	}
	return float64(st.Rest.TotalS) / 60
}

// GDD §3.2: ≤25 min -> 5 min of rest, 26–45 -> 10, 46 or more -> 15.
func TestRestLengthFollowsThePomodoroLength(t *testing.T) {
	for _, c := range []struct {
		crop string
		min  int
		rest float64
	}{{"daisy", 0, 5}, {"tomato", 0, 5}, {"sunflower", 0, 10}, {"apple", 0, 10}, {"oak", 60, 15}, {"oak", 90, 15}, {"oak", 120, 15}} {
		e := newFreshEnv(t)
		unlock(t, e, c.crop)
		st := finishAndHarvest(t, e, c.crop, c.min, time.Minute)
		if got := restMinutes(st); got != c.rest {
			t.Errorf("%s %d min: rest %.0f min, GDD %.0f", c.crop, c.min, got, c.rest)
		}
		if st.Rest != nil && (st.Rest.RemainingMs > st.Rest.TotalS*1000 || st.Rest.RemainingMs <= 0) {
			t.Errorf("%s: remaining %d ms of %d s", c.crop, st.Rest.RemainingMs, st.Rest.TotalS)
		}
	}
}

func TestRestCountsDownAndEndsByItself(t *testing.T) {
	e := newFreshEnv(t)
	st := finishAndHarvest(t, e, "daisy", 0, 0)
	if st.Rest == nil || st.Rest.RemainingMs != 5*60*1000 {
		t.Fatalf("rest = %+v", st.Rest)
	}
	e.clock.Advance(2 * time.Minute)
	if got := e.state().Rest.RemainingMs; got != 3*60*1000 {
		t.Fatalf("after 2 min: %d ms", got)
	}
	e.clock.Advance(3 * time.Minute)
	if e.state().Rest != nil {
		t.Fatal("the rest should be over")
	}
}

func TestSkippingTheRest(t *testing.T) {
	e := newFreshEnv(t)
	finishAndHarvest(t, e, "daisy", 0, 0)
	e.expect(200, "POST", "/api/rest/skip", nil)
	if e.state().Rest != nil {
		t.Fatal("rest still running after skipping")
	}
	if code := errCode(e.expect(409, "POST", "/api/rest/skip", nil)); code != "no_rest" {
		t.Fatalf("skipping with no rest: %s", code)
	}
}

func TestStartingAPomodoroEndsTheRest(t *testing.T) {
	e := newFreshEnv(t)
	finishAndHarvest(t, e, "daisy", 0, 0)
	e.expect(200, "POST", "/api/plots/1/clear", map[string]bool{"confirm": true})
	e.expect(201, "POST", "/api/pomodoros", service.PlantRequest{PlotID: 1, PlantType: "daisy"})
	if e.state().Rest != nil {
		t.Fatal("rest should end when the next Pomodoro starts")
	}
}

// Decision: a rest is only offered when the player harvests soon after the Pomodoro ended (15 min).
func TestNoRestWhenHarvestingLongAfterFinishing(t *testing.T) {
	e := newFreshEnv(t)
	if st := finishAndHarvest(t, e, "daisy", 0, 14*time.Minute); st.Rest == nil {
		t.Fatal("14 min after finishing the rest should still be offered")
	}
	e.expect(200, "POST", "/api/rest/skip", nil)
	e.expect(200, "POST", "/api/plots/1/clear", map[string]bool{"confirm": true})
	if st := finishAndHarvest(t, e, "daisy", 0, 16*time.Minute); st.Rest != nil {
		t.Fatalf("16 min after finishing, the player already had their break: %+v", st.Rest)
	}
}

func TestAPlantThatWitheredWhileAwayNeverGivesARest(t *testing.T) {
	e := newFreshEnv(t)
	e.expect(201, "POST", "/api/pomodoros", service.PlantRequest{PlotID: 1, PlantType: "daisy"})
	e.clock.Advance(30 * time.Hour)
	e.expect(200, "POST", "/api/plots/1/harvest", nil)
	if e.state().Rest != nil {
		t.Fatal("a rest after a plant that wilted while the player was away makes no sense")
	}
}

func TestRestCanBeSwitchedOff(t *testing.T) {
	e := newFreshEnv(t)
	e.expect(200, "POST", "/api/settings", map[string]string{"key": "rest_enabled", "value": "0"})
	if st := finishAndHarvest(t, e, "daisy", 0, 0); st.Rest != nil {
		t.Fatalf("rest started although they are switched off: %+v", st.Rest)
	}
	e.expect(200, "POST", "/api/plots/1/clear", map[string]bool{"confirm": true})
	e.expect(200, "POST", "/api/settings", map[string]string{"key": "rest_enabled", "value": "1"})
	if st := finishAndHarvest(t, e, "daisy", 0, 0); st.Rest == nil {
		t.Fatal("rest should be back after switching it on")
	}
}

func TestRestLengthsAreConfigurablePerBucket(t *testing.T) {
	e := newFreshEnv(t)
	unlock(t, e, "sunflower")
	e.expect(200, "POST", "/api/settings", map[string]string{"key": "rest_short_min", "value": "2"})
	e.expect(200, "POST", "/api/settings", map[string]string{"key": "rest_medium_min", "value": "20"})
	if got := restMinutes(finishAndHarvest(t, e, "daisy", 0, 0)); got != 2 {
		t.Fatalf("short bucket: %.0f min, configured 2", got)
	}
	e.expect(200, "POST", "/api/rest/skip", nil)
	e.expect(200, "POST", "/api/plots/1/clear", map[string]bool{"confirm": true})
	if got := restMinutes(finishAndHarvest(t, e, "sunflower", 0, 0)); got != 20 {
		t.Fatalf("medium bucket: %.0f min, configured 20", got)
	}
	e.expect(200, "POST", "/api/rest/skip", nil)
	e.expect(200, "POST", "/api/plots/1/clear", map[string]bool{"confirm": true})
	unlock(t, e, "oak")
	if got := restMinutes(finishAndHarvest(t, e, "oak", 60, 0)); got != 15 {
		t.Fatalf("long bucket was not configured: %.0f min, GDD default 15", got)
	}
}

func TestRestSettingsAreValidated(t *testing.T) {
	e := newFreshEnv(t)
	for _, c := range []struct{ key, value string }{
		{"rest_short_min", "0"}, {"rest_short_min", "61"}, {"rest_long_min", "abc"}, {"rest_medium_min", "-3"},
		{"rest_short_min", "5.5"}, {"rest_short_min", ""}, {"rest_enabled", "yes"}, {"rest_enabled", "2"},
	} {
		if code := errCode(e.expect(400, "POST", "/api/settings", map[string]string{"key": c.key, "value": c.value})); code != "invalid_request" {
			t.Errorf("%s=%q: %s", c.key, c.value, code)
		}
	}
	for _, c := range []struct{ key, value string }{{"rest_short_min", "1"}, {"rest_short_min", "60"}, {"rest_enabled", "0"}, {"rest_enabled", "1"}} {
		e.expect(200, "POST", "/api/settings", map[string]string{"key": c.key, "value": c.value})
	}
}

func TestAnInvalidStoredRestLengthFallsBackToTheDefault(t *testing.T) {
	e := newFreshEnv(t)
	dbExec(t, e, `INSERT INTO settings (player_id,key,value) VALUES (1,'rest_short_min','999')`) // e.g. a corrupted row
	if got := restMinutes(finishAndHarvest(t, e, "daisy", 0, 0)); got != 5 {
		t.Fatalf("rest = %.0f min, want the default 5", got)
	}
}

func TestTwoDevicesSeeTheSameRest(t *testing.T) {
	e := newFreshEnv(t)
	finishAndHarvest(t, e, "daisy", 0, 0)
	e.clock.Advance(90 * time.Second)
	var a, b service.State
	json.Unmarshal(e.expect(200, "GET", "/api/state", nil), &a)
	json.Unmarshal(e.expect(200, "GET", "/api/state", nil), &b)
	if a.Rest == nil || b.Rest == nil || a.Rest.EndsAt != b.Rest.EndsAt || a.Rest.RemainingMs != 210_000 {
		t.Fatalf("device A %+v, device B %+v", a.Rest, b.Rest)
	}
}
