package api

import (
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/luk224/pomofarm/backend/internal/game"
	"github.com/luk224/pomofarm/backend/internal/service"
)

func unlock(t *testing.T, e *env, keys ...string) {
	t.Helper()
	for _, k := range keys {
		dbExec(t, e, `INSERT OR IGNORE INTO unlocks (player_id,kind,key,at) VALUES (1,'seed',?,'t')`, k)
	}
}

type lifecycle struct {
	planned          time.Duration
	reward           int
	life             time.Duration
	rateMilliPerHour int64
}

// grow plants the seed, lets it mature, harvests it and reports what the server decided.
func grow(t *testing.T, e *env, plant string, minutes int) lifecycle {
	t.Helper()
	e.expect(201, "POST", "/api/pomodoros", service.PlantRequest{PlotID: 1, PlantType: plant, DurationMin: minutes})
	planned := time.Duration(e.state().Pomodoro.PlannedS) * time.Second
	e.clock.Advance(planned)
	var out struct {
		Reward int           `json:"reward_focus"`
		State  service.State `json:"state"`
	}
	json.Unmarshal(e.expect(200, "POST", "/api/plots/1/harvest", nil), &out)
	p := out.State.Plots[0]
	matured, _ := time.Parse(time.RFC3339Nano, *p.MaturedAt)
	wilts, _ := time.Parse(time.RFC3339Nano, *p.WiltsAt)
	return lifecycle{planned, out.Reward, wilts.Sub(matured), out.State.Silo.RateMilliPerHour}
}

func clearPlot1(e *env) {
	e.expect(200, "POST", "/api/plots/1/clear", map[string]bool{"confirm": true})
}

// GDD §4.3, Modo Flow: 60/75/90/105/120 min -> 25/35/46/58/71 💧 and 108/135/162/189/216 h of life.
func TestFlowTableOfTheGDD(t *testing.T) {
	e := newFreshEnv(t)
	unlock(t, e, "oak")
	for _, c := range []struct {
		min, reward, lifeH int
		yield              float64
	}{{60, 25, 108, 503}, {75, 35, 135, 752}, {90, 46, 162, 1044}, {105, 58, 189, 1378}, {120, 71, 216, 1752}} {
		got := grow(t, e, "oak", c.min)
		if got.planned != time.Duration(c.min)*time.Minute {
			t.Errorf("%d min: planned %v", c.min, got.planned)
		}
		if got.reward != c.reward {
			t.Errorf("%d min: reward %d 💧, GDD %d", c.min, got.reward, c.reward)
		}
		if got.life != time.Duration(c.lifeH)*time.Hour {
			t.Errorf("%d min: life %v, GDD %d h", c.min, got.life, c.lifeH)
		}
		// the production rate follows from the yield per cycle: Y / life (🪙/h × 1000)
		wantRate := c.yield / float64(c.lifeH) * 1000
		if math.Abs(float64(got.rateMilliPerHour)-wantRate) > wantRate*0.01 {
			t.Errorf("%d min: rate %d, want ≈%.0f (GDD yield %.0f 🪙 over %d h)", c.min, got.rateMilliPerHour, wantRate, c.yield, c.lifeH)
		}
		clearPlot1(e)
	}
}

// The GDD defines life as 108 h · d/60 with no steps: a 97-minute session lives 174.6 h, not 174.
func TestFlowLifeAndRewardAreContinuousBetweenTheTableRows(t *testing.T) {
	e := newFreshEnv(t)
	unlock(t, e, "oak")
	for _, min := range []int{61, 67, 82, 97, 113, 119} {
		got := grow(t, e, "oak", min)
		wantLife := time.Duration(float64(108*time.Hour) * float64(min) / 60)
		if d := got.life - wantLife; d < -time.Second || d > time.Second {
			t.Errorf("%d min: life %v, want %v", min, got.life, wantLife)
		}
		if want := int(math.Round(25 * math.Pow(float64(min)/60, 1.5))); got.reward != want {
			t.Errorf("%d min: reward %d, want %d", min, got.reward, want)
		}
		clearPlot1(e)
	}
}

func TestFlowRangeIsEnforcedByTheServer(t *testing.T) {
	e := newFreshEnv(t)
	unlock(t, e, "oak", "apple")
	for _, min := range []int{59, 121, -5, 600} {
		if code := errCode(e.expect(400, "POST", "/api/pomodoros", service.PlantRequest{PlotID: 1, PlantType: "oak", DurationMin: min})); code != "invalid_request" {
			t.Errorf("%d min: %s", min, code)
		}
	}
	// only the Oak is flexible
	e.expect(400, "POST", "/api/pomodoros", service.PlantRequest{PlotID: 1, PlantType: "apple", DurationMin: 60})
	// no duration given: the Oak defaults to its 60 minutes
	e.expect(201, "POST", "/api/pomodoros", service.PlantRequest{PlotID: 1, PlantType: "oak"})
	if got := e.state().Pomodoro.PlannedS; got != 3600 {
		t.Fatalf("default oak = %d s", got)
	}
}

func TestAppleAndOakHaveTheirGDDStats(t *testing.T) {
	e := newFreshEnv(t)
	unlock(t, e, "apple")
	a := grow(t, e, "apple", 0)
	if a.planned != 45*time.Minute || a.reward != 15 || a.life != 72*time.Hour {
		t.Fatalf("apple: %+v, GDD 45 min / 15 💧 / 72 h", a)
	}
}

func TestApplesAndOaksCostWhatTheGDDSays(t *testing.T) {
	e := newFreshEnv(t)
	giveFocus(t, e, 99)
	if code := errCode(e.expect(409, "POST", "/api/unlocks", map[string]string{"kind": "seed", "key": "apple"})); code != "insufficient_focus" {
		t.Fatalf("99 💧 for the apple: %s", code)
	}
	giveFocus(t, e, 100)
	e.expect(201, "POST", "/api/unlocks", map[string]string{"kind": "seed", "key": "apple"})
	if got := e.state().Player.FocusPoints; got != 0 {
		t.Fatalf("after the apple: %d 💧", got)
	}
	giveFocus(t, e, 250)
	e.expect(201, "POST", "/api/unlocks", map[string]string{"kind": "seed", "key": "oak"})
	if got := e.state().Player.FocusPoints; got != 0 {
		t.Fatalf("after the oak: %d 💧", got)
	}
}

func TestFlowTableEndpoint(t *testing.T) {
	e := newFreshEnv(t)
	var rows []game.FlowRow
	json.Unmarshal(e.expect(200, "GET", "/api/flow", nil), &rows)
	if len(rows) != 61 || rows[0].DurationMin != 60 || rows[60].DurationMin != 120 {
		t.Fatalf("%d rows, first %v last %v", len(rows), rows[0], rows[len(rows)-1])
	}
	byMin := map[int]game.FlowRow{}
	for i, r := range rows {
		byMin[r.DurationMin] = r
		if i > 0 && (r.Reward < rows[i-1].Reward || r.LifeH <= rows[i-1].LifeH || r.Yield <= rows[i-1].Yield) {
			t.Fatalf("row %d is not increasing: %+v after %+v", i, r, rows[i-1])
		}
	}
	for min, want := range map[int][3]float64{60: {25, 108, 503}, 75: {35, 135, 752}, 90: {46, 162, 1044}, 105: {58, 189, 1378}, 120: {71, 216, 1752}} {
		r := byMin[min]
		if float64(r.Reward) != want[0] || math.Abs(r.LifeH-want[1]) > 1e-9 || math.Abs(r.Yield-want[2]) > 0.6 {
			t.Errorf("%d min: %+v, GDD reward %v life %v yield %v", min, r, want[0], want[1], want[2])
		}
	}
	if byMin[60].RestMin != 15 || byMin[120].RestMin != 15 {
		t.Errorf("Flow sessions are ≥46 min: rest should be 15, got %d / %d", byMin[60].RestMin, byMin[120].RestMin)
	}
}
