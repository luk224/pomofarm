package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/luk224/pomofarm/backend/internal/game"
	"github.com/luk224/pomofarm/backend/internal/service"
)

// GDD §4.2: "🪙/día = (minutos de foco al día) × e(d) × M". With a farm whose neighbours give no bonus (M = 1) a
// Normal player (3 h/day in 45-minute Pomodoros: 4 apples a day) must earn exactly 180 · e(45) = 1199.1 🪙/day once the
// farm is in steady state. This checks the whole stack (plant → mature → live → wilt → clear → replant → collect).
func TestSteadyStateIncomeMatchesTheGDDFormula(t *testing.T) {
	e := newFreshEnv(t)
	giveFocus(t, e, 3000)
	for i := 0; i < 15; i++ {
		e.expect(201, "POST", "/api/plots", nil)
	}
	for i := 0; i < 4; i++ {
		e.expect(200, "POST", "/api/silo/upgrade", nil)
	}
	unlock(t, e, "apple")
	plots := e.state().Plots

	perDay := []float64{}
	next := 0
	for day := 1; day <= 30; day++ {
		for i := 0; i < 4; i++ { // four 45-minute Pomodoros: 3 h of focus, rotating over 16 plots (a plot is reused after 4 days)
			p := plots[next%16]
			next++
			for _, cur := range e.state().Plots {
				if cur.ID == p.ID && cur.State != "empty" { // plots are listed by position, so look it up by id
					e.expect(200, "POST", fmt.Sprintf("/api/plots/%d/clear", p.ID), map[string]bool{"confirm": true})
				}
			}
			e.expect(201, "POST", "/api/pomodoros", service.PlantRequest{PlotID: p.ID, PlantType: "apple"})
			e.clock.Advance(45 * time.Minute)
			e.expect(200, "POST", fmt.Sprintf("/api/plots/%d/harvest", p.ID), nil)
			e.expect(200, "POST", "/api/rest/skip", nil) // not relevant here
			e.clock.Advance(15 * time.Minute)
		}
		e.clock.Advance(20 * time.Hour) // the rest of the day
		before := e.state().Player.CoinsMilli
		e.expect(200, "POST", "/api/silo/collect", nil)
		perDay = append(perDay, float64(e.state().Player.CoinsMilli-before)/1000)
	}
	var sum float64
	for _, v := range perDay[11:] { // steady state: after the first plants have lived their 72 h
		sum += v
	}
	got := sum / float64(len(perDay)-11)
	want := 180 * game.YieldPerMin(45) // 1199.15
	if math.Abs(got-want) > want*0.01 {
		t.Fatalf("steady-state income %.1f 🪙/day, GDD formula %.1f (days 12–30: %v)", got, want, perDay[11:])
	}
	t.Logf("steady-state income: %.1f 🪙/day vs GDD %.1f (diff %.2f%%)", got, want, (got-want)/want*100)
}

// GDD §8: "Ausencia de 90 días: se ingresa como máximo lo que permite el Silo; ninguna planta produce más allá de su vida útil".
func TestNinetyDaysAwayWithAFullFarm(t *testing.T) {
	e := newFreshEnv(t)
	giveFocus(t, e, 3000)
	for i := 0; i < 15; i++ {
		e.expect(201, "POST", "/api/plots", nil)
	}
	e.expect(200, "POST", "/api/silo/upgrade", nil)
	e.expect(200, "POST", "/api/silo/upgrade", nil) // Silo level 2: 36 h
	crops := []string{"daisy", "tomato", "sunflower", "apple", "oak"}
	var lifetimeMax float64
	for i, p := range e.state().Plots {
		c, _ := game.CropByKey(crops[i%5])
		mature(t, e, int(p.ID), crops[i%5], 0, c.LifeH)
		lifetimeMax += game.CycleYield(float64(c.DurationMin)) * game.PlotMultiplierCap
	}
	e.clock.Advance(90 * 24 * time.Hour)
	st := e.state()
	cap := float64(st.Silo.CapacityMilli) / 1000
	got := float64(st.Silo.ContentMilli) / 1000
	if got > cap+0.01 {
		t.Fatalf("Silo holds %.1f 🪙, more than its capacity %.1f", got, cap)
	}
	if got > lifetimeMax {
		t.Fatalf("Silo holds %.1f 🪙, more than every plant could ever make (%.1f)", got, lifetimeMax)
	}
	if st.Silo.RateMilliPerHour != 0 {
		t.Fatalf("after 90 days still producing %d/h", st.Silo.RateMilliPerHour)
	}
	for _, p := range st.Plots {
		if p.State != "withered" || p.Bonus != nil {
			t.Fatalf("plot %d after 90 days: %s bonus %v", p.ID, p.State, p.Bonus)
		}
	}
	e.expect(200, "POST", "/api/silo/collect", nil)
	e.clock.Advance(90 * 24 * time.Hour)
	if silo, _ := milli(t, e); silo != 0 {
		t.Fatalf("withered plants produced %d more after 90 further days", silo)
	}
	t.Logf("90 days away: Silo (36 h) held %.1f of at most %.1f 🪙", got, cap)
}

type garbage struct {
	method, path, body string
}

// No input must ever make the server answer 5xx, crash, or leak internals.
func TestNoEndpointAnswersWithAServerErrorOnGarbage(t *testing.T) {
	e := newFreshEnv(t)
	bodies := []string{
		"", "{", "}", "[]", "null", "true", "0", `"x"`, `{"plot_id":"x"}`, `{"plot_id":-1}`, `{"plot_id":1e300}`, `{"plot_id":99999999999999999999999}`,
		`{"plant_type":"` + strings.Repeat("a", 5000) + `"}`, `{"plot_id":1,"plant_type":"daisy","duration_min":"x"}`,
		`{"plot_id":1,"plant_type":"daisy","tag":"` + strings.Repeat("é", 500) + `"}`, `{"kind":"seed","key":null}`, `{"key":"rest_short_min","value":null}`,
		`{"confirm":"yes"}`, `{"confirm":true,"confirm":false}`, `{"confirm":1}`, `{"confirm":null}`, `{"confirm":[true]}`, `{"kind":"hive","plot_id":"x"}`, `{"kind":"dog","plot_id":-1}`, `{"kind":null}`, `{"kind":"path","x":"a","y":null}`,
		`{"kind":"lantern","x":1e300,"y":-1e300}`, `{"kind":"path","x":99999999999999999999999,"y":1}`, `{"kind":"hat","x":3}`, `{"kind":"` + strings.Repeat("z", 3000) + `","x":4,"y":4}`,
		`{"kind":"animal","key":"` + strings.Repeat("k", 3000) + `"}`, `{"kind":"animal","key":null}`, `{"x":-2147483649,"y":2147483648,"kind":"path"}`, "\x00\x01\x02", `{"a":` + strings.Repeat("[", 200) + strings.Repeat("]", 200) + `}`, `{"plot_id":1,"plot_id":2}`,
	}
	paths := []struct{ method, path string }{
		{"POST", "/api/pomodoros"}, {"POST", "/api/pomodoros/active/pause"}, {"POST", "/api/pomodoros/active/resume"}, {"POST", "/api/pomodoros/active/cancel"},
		{"POST", "/api/plots"}, {"POST", "/api/plots/1/harvest"}, {"POST", "/api/plots/1/clear"}, {"POST", "/api/plots/abc/harvest"}, {"POST", "/api/plots/-1/clear"},
		{"POST", "/api/plots/99999999999999999999/harvest"}, {"POST", "/api/plots/0/harvest"}, {"POST", "/api/unlocks"}, {"POST", "/api/settings"},
		{"POST", "/api/silo/collect"}, {"POST", "/api/silo/upgrade"}, {"POST", "/api/rest/skip"},
		{"POST", "/api/structures"}, {"POST", "/api/structures/1/move"}, {"POST", "/api/structures/abc/move"}, {"POST", "/api/structures/-1/move"},
		{"POST", "/api/structures/99999999999999999999/move"}, {"POST", "/api/decor"}, {"POST", "/api/decor/1/move"}, {"POST", "/api/decor/abc/move"},
		{"POST", "/api/prestige"}, {"GET", "/api/prestige"}, {"GET", "/api/stats"}, {"GET", "/api/book"}, {"GET", "/api/book/export.csv"}, {"POST", "/api/book"},
		{"GET", "/api/book?month=%00"}, {"GET", "/api/book?month=2026-10&tz=%ff%fe"}, {"GET", "/api/book?month=" + strings.Repeat("9", 500)}, {"GET", "/api/book?month=../../etc/passwd"},
		{"GET", "/api/book?tz=" + strings.Repeat("Europe/", 100)}, {"GET", "/api/stats?tz=%00"}, {"GET", "/api/stats?tz=../../etc/localtime"}, {"GET", "/api/stats?tz=UTC&tz=Mars/Base"},
		{"GET", "/api/book/export.csv?tz=%00"}, {"GET", "/api/book/export.csv?tz=" + strings.Repeat("a", 2000)}, {"GET", "/api/book?month=0000-00"}, {"GET", "/api/book?month=9999-12"},
		{"DELETE", "/api/decor/1"}, {"DELETE", "/api/decor/abc"}, {"DELETE", "/api/decor/-5"}, {"DELETE", "/api/decor"}, {"GET", "/api/decor"},
		{"GET", "/api/state"}, {"GET", "/api/flow"}, {"GET", "/api/health"}, {"GET", "/api/nope"}, {"PUT", "/api/state"}, {"DELETE", "/api/plots"}, {"PATCH", "/api/pomodoros"},
		{"GET", "/api/plots/1/harvest"}, {"POST", "/api/state"}, {"GET", "/api/../etc/passwd"}, {"GET", "/api/state?x=%00%ff"},
	}
	for _, p := range paths {
		for _, b := range bodies {
			req := httptest.NewRequest(p.method, p.path, bytes.NewBufferString(b))
			req.Header.Set("Content-Type", "application/json")
			resp, err := e.app.Test(req, -1)
			if err != nil {
				t.Fatalf("%s %s %q: transport error %v", p.method, p.path, truncate(b), err)
			}
			var out bytes.Buffer
			out.ReadFrom(resp.Body)
			if resp.StatusCode >= 500 {
				t.Fatalf("%s %s with body %q answered %d: %s", p.method, p.path, truncate(b), resp.StatusCode, out.String())
			}
			if resp.StatusCode >= 400 && resp.StatusCode != 404 && resp.StatusCode != 405 && !json.Valid(out.Bytes()) {
				t.Errorf("%s %s %q: %d with a non-JSON body: %.60s", p.method, p.path, truncate(b), resp.StatusCode, out.String())
			}
			low := strings.ToLower(out.String())
			if strings.Contains(low, "sql") || strings.Contains(low, "goroutine") || strings.Contains(low, ".go:") {
				t.Errorf("%s %s %q leaks internals: %.100s", p.method, p.path, truncate(b), out.String())
			}
		}
	}
	// the game itself is untouched by all that noise
	if st := e.state(); st.Pomodoro != nil && st.Player.FocusPoints > 0 {
		t.Logf("state after the noise: %+v", st.Player)
	}
}

func truncate(s string) string {
	if len(s) > 40 {
		return s[:40] + "…"
	}
	return s
}

func TestOversizedBodiesAreRefused(t *testing.T) {
	e := newFreshEnv(t)
	big := `{"plot_id":1,"plant_type":"daisy","tag":"` + strings.Repeat("x", 40_000) + `"}`
	req := httptest.NewRequest("POST", "/api/pomodoros", bytes.NewBufferString(big))
	req.Header.Set("Content-Type", "application/json")
	resp, err := e.app.Test(req, -1)
	// Fiber's in-memory test transport reports an oversized body as an error; over real HTTP it is a 413
	// (checked against the running server in the QA report).
	if err == nil && resp.StatusCode != 413 {
		t.Fatalf("a 40 KB body answered %d, want 413", resp.StatusCode)
	}
	if err != nil && !strings.Contains(err.Error(), "body size exceeds") {
		t.Fatalf("unexpected error: %v", err)
	}
	if e.state().Pomodoro != nil {
		t.Fatal("an oversized request started a Pomodoro")
	}
}
