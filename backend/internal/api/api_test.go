package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/luk224/pomofarm/backend/internal/game"
	"github.com/luk224/pomofarm/backend/internal/service"
	"github.com/luk224/pomofarm/backend/internal/store"
)

type env struct {
	t     *testing.T
	app   *fiber.App
	db    *sql.DB
	clock *game.FakeClock
}

func newEnv(t *testing.T, plots int) *env {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec(`INSERT INTO players (id,name,created_at,last_seen_at) VALUES (1,'luk','t','t')`); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < plots; i++ {
		if _, err := db.Exec(`INSERT INTO plots (player_id,x,y) VALUES (1,?,0)`, i); err != nil {
			t.Fatal(err)
		}
	}
	clock := &game.FakeClock{T: time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)}
	return &env{t: t, app: New(db, clock), db: db, clock: clock}
}

func dbExec(t *testing.T, e *env, q string, args ...any) {
	t.Helper()
	if _, err := e.db.Exec(q, args...); err != nil {
		t.Fatal(err)
	}
}

func (e *env) do(method, path string, body any) (int, []byte) {
	e.t.Helper()
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	resp, err := e.app.Test(req, -1)
	if err != nil {
		e.t.Fatal(err)
	}
	var out bytes.Buffer
	out.ReadFrom(resp.Body)
	return resp.StatusCode, out.Bytes()
}

func (e *env) state() service.State {
	e.t.Helper()
	code, b := e.do("GET", "/api/state", nil)
	if code != 200 {
		e.t.Fatalf("state: %d %s", code, b)
	}
	var st service.State
	if err := json.Unmarshal(b, &st); err != nil {
		e.t.Fatal(err)
	}
	return st
}

func (e *env) expect(code int, method, path string, body any) []byte {
	e.t.Helper()
	got, b := e.do(method, path, body)
	if got != code {
		e.t.Fatalf("%s %s = %d (%s), want %d", method, path, got, b, code)
	}
	return b
}

func errCode(b []byte) string {
	var m map[string]string
	json.Unmarshal(b, &m)
	return m["error"]
}

func TestHealthReportsSchema(t *testing.T) {
	e := newEnv(t, 0)
	var body struct {
		Status        string `json:"status"`
		SchemaVersion int    `json:"schema_version"`
	}
	json.Unmarshal(e.expect(200, "GET", "/api/health", nil), &body)
	if body.Status != "ok" || body.SchemaVersion != 2 {
		t.Fatalf("body = %+v", body)
	}
}

func TestNoPlayerIs404(t *testing.T) {
	db, _ := store.Open(filepath.Join(t.TempDir(), "t.db"))
	defer db.Close()
	e := &env{t: t, app: New(db, &game.FakeClock{T: time.Now()}), db: db}
	if c := errCode(e.expect(404, "GET", "/api/state", nil)); c != "no_player" {
		t.Fatalf("code = %q", c)
	}
}

func TestPlantRunsAndCountsDown(t *testing.T) {
	e := newEnv(t, 2)
	e.expect(201, "POST", "/api/pomodoros", service.PlantRequest{PlotID: 1, PlantType: "daisy", Tag: "email"})
	st := e.state()
	if st.Pomodoro == nil || st.Pomodoro.Status != "running" || st.Pomodoro.RemainingMs != 600_000 {
		t.Fatalf("pomodoro = %+v", st.Pomodoro)
	}
	if st.Plots[0].State != "growing" {
		t.Fatalf("plot = %+v", st.Plots[0])
	}
	e.clock.Advance(4 * time.Minute)
	if got := e.state().Pomodoro.RemainingMs; got != 360_000 {
		t.Fatalf("remaining after 4 min = %d", got)
	}
}

// "Abrir en dos dispositivos": the second one sees the existing Pomodoro and cannot start another.
func TestSecondDeviceCannotStartAnother(t *testing.T) {
	e := newEnv(t, 2)
	e.expect(201, "POST", "/api/pomodoros", service.PlantRequest{PlotID: 1, PlantType: "daisy"})
	b := e.expect(409, "POST", "/api/pomodoros", service.PlantRequest{PlotID: 2, PlantType: "daisy"})
	if errCode(b) != "pomodoro_active" {
		t.Fatalf("code = %s", b)
	}
	if st := e.state(); st.Pomodoro == nil || *st.Pomodoro.PlotID != 1 {
		t.Fatalf("existing pomodoro lost: %+v", st.Pomodoro)
	}
}

func TestConcurrentPlantOnlyOneWins(t *testing.T) {
	e := newEnv(t, 8)
	var wg sync.WaitGroup
	codes := make([]int, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			codes[i], _ = e.do("POST", "/api/pomodoros", service.PlantRequest{PlotID: int64(i + 1), PlantType: "daisy"})
		}(i)
	}
	wg.Wait()
	created, conflict := 0, 0
	for _, c := range codes {
		switch c {
		case 201:
			created++
		case 409:
			conflict++
		}
	}
	if created != 1 || conflict != 7 {
		t.Fatalf("codes = %v, want exactly one 201 and seven 409", codes)
	}
}

func TestPauseResumeFreezesTime(t *testing.T) {
	e := newEnv(t, 1)
	e.expect(201, "POST", "/api/pomodoros", service.PlantRequest{PlotID: 1, PlantType: "daisy"})
	e.clock.Advance(2 * time.Minute)
	e.expect(200, "POST", "/api/pomodoros/active/pause", nil)
	e.clock.Advance(48 * time.Hour)
	st := e.state()
	if st.Pomodoro.Status != "paused" || st.Pomodoro.RemainingMs != 480_000 {
		t.Fatalf("paused state = %+v", st.Pomodoro)
	}
	if code := errCode(e.expect(409, "POST", "/api/pomodoros/active/pause", nil)); code != "wrong_state" {
		t.Fatalf("double pause code = %s", code)
	}
	e.expect(200, "POST", "/api/pomodoros/active/resume", nil)
	e.clock.Advance(3 * time.Minute)
	if got := e.state().Pomodoro.RemainingMs; got != 300_000 {
		t.Fatalf("remaining = %d, want 300000", got)
	}
}

func TestCancelClearsPlotAndGivesNothing(t *testing.T) {
	e := newEnv(t, 1)
	e.expect(201, "POST", "/api/pomodoros", service.PlantRequest{PlotID: 1, PlantType: "daisy"})
	e.expect(200, "POST", "/api/pomodoros/active/cancel", nil)
	st := e.state()
	if st.Pomodoro != nil || st.Plots[0].State != "empty" || st.Player.FocusPoints != 0 {
		t.Fatalf("after cancel: %+v", st)
	}
	if code := errCode(e.expect(409, "POST", "/api/pomodoros/active/pause", nil)); code != "no_active_pomodoro" {
		t.Fatalf("code = %s", code)
	}
	e.expect(201, "POST", "/api/pomodoros", service.PlantRequest{PlotID: 1, PlantType: "daisy"}) // plot reusable
}

func TestFullCycleHarvestOnce(t *testing.T) {
	e := newEnv(t, 1)
	e.expect(201, "POST", "/api/pomodoros", service.PlantRequest{PlotID: 1, PlantType: "daisy"})
	if code := errCode(e.expect(409, "POST", "/api/plots/1/harvest", nil)); code != "not_mature" {
		t.Fatalf("early harvest code = %s", code)
	}
	e.clock.Advance(10*time.Minute - time.Second)
	if e.state().Pomodoro == nil {
		t.Fatal("pomodoro finished 1 s early")
	}
	e.clock.Advance(time.Second)
	st := e.state()
	if st.Pomodoro != nil || st.Plots[0].State != "mature" || st.Plots[0].WiltsAt == nil {
		t.Fatalf("after time is up: %+v", st)
	}
	var out struct {
		Reward int           `json:"reward_focus"`
		State  service.State `json:"state"`
	}
	json.Unmarshal(e.expect(200, "POST", "/api/plots/1/harvest", nil), &out)
	if out.Reward != 1 || out.State.Player.FocusPoints != 1 || out.State.Player.LifetimeFocus != 1 {
		t.Fatalf("harvest = %+v", out)
	}
	if code := errCode(e.expect(409, "POST", "/api/plots/1/harvest", nil)); code != "already_harvested" {
		t.Fatalf("second harvest code = %s", code)
	}
	if !out.State.Plots[0].Harvested && !e.state().Plots[0].Harvested {
		t.Fatal("plot not marked harvested")
	}
}

// The browser can be closed for days: completion is recorded at the true finish time, not when noticed.
func TestCompletionTimeIsExactAfterLongAbsence(t *testing.T) {
	e := newEnv(t, 1)
	e.expect(201, "POST", "/api/pomodoros", service.PlantRequest{PlotID: 1, PlantType: "daisy"})
	start := e.clock.T
	e.clock.Advance(72 * time.Hour)
	st := e.state()
	want := start.Add(10 * time.Minute).UTC().Format(time.RFC3339Nano)
	if st.Plots[0].MaturedAt == nil || *st.Plots[0].MaturedAt != want {
		t.Fatalf("matured_at = %v, want %s", st.Plots[0].MaturedAt, want)
	}
}

func TestValidation(t *testing.T) {
	e := newEnv(t, 1)
	for name, c := range map[string]struct {
		code int
		req  service.PlantRequest
		err  string
	}{
		"unknown plant":     {400, service.PlantRequest{PlotID: 1, PlantType: "potato"}, "invalid_request"},
		"wrong duration":    {400, service.PlantRequest{PlotID: 1, PlantType: "daisy", DurationMin: 30}, "invalid_request"},
		"flow out of range": {400, service.PlantRequest{PlotID: 1, PlantType: "oak", DurationMin: 130}, "invalid_request"},
		"locked seed":       {403, service.PlantRequest{PlotID: 1, PlantType: "tomato"}, "seed_locked"},
		"unknown plot":      {404, service.PlantRequest{PlotID: 99, PlantType: "daisy"}, "not_found"},
	} {
		if got := errCode(e.expect(c.code, "POST", "/api/pomodoros", c.req)); got != c.err {
			t.Errorf("%s: code %q, want %q", name, got, c.err)
		}
	}
	e.expect(400, "POST", "/api/pomodoros", "not an object")
	e.expect(400, "POST", "/api/plots/abc/harvest", nil)
	if e.state().Pomodoro != nil {
		t.Fatal("a rejected request started a pomodoro")
	}
}

func TestOccupiedPlotRejected(t *testing.T) {
	e := newEnv(t, 1)
	e.expect(201, "POST", "/api/pomodoros", service.PlantRequest{PlotID: 1, PlantType: "daisy"})
	e.clock.Advance(11 * time.Minute) // mature, plot still occupied by the standing plant
	if code := errCode(e.expect(409, "POST", "/api/pomodoros", service.PlantRequest{PlotID: 1, PlantType: "daisy"})); code != "plot_busy" {
		t.Fatalf("code = %s", code)
	}
}

func TestUnlockedSeedAndFlowReward(t *testing.T) {
	e := newEnv(t, 1)
	dbExec(t, e, `INSERT INTO unlocks (player_id,kind,key,at) VALUES (1,'seed','oak','t')`)
	e.expect(201, "POST", "/api/pomodoros", service.PlantRequest{PlotID: 1, PlantType: "oak", DurationMin: 90})
	if got := e.state().Pomodoro.PlannedS; got != 5400 {
		t.Fatalf("planned = %d", got)
	}
	e.clock.Advance(90 * time.Minute)
	var out struct {
		Reward int `json:"reward_focus"`
	}
	json.Unmarshal(e.expect(200, "POST", "/api/plots/1/harvest", nil), &out)
	if out.Reward != game.FlowReward(90) || out.Reward != 46 {
		t.Fatalf("flow reward = %d, want 46", out.Reward)
	}
	wilts, _ := time.Parse(time.RFC3339Nano, *e.state().Plots[0].WiltsAt)
	matured, _ := time.Parse(time.RFC3339Nano, *e.state().Plots[0].MaturedAt)
	if wilts.Sub(matured) != time.Duration(game.FlowLifeH(90))*time.Hour {
		t.Fatalf("life = %v", wilts.Sub(matured))
	}
}

// newFreshEnv is a brand-new installation: no player until EnsurePlayer runs.
func newFreshEnv(t *testing.T) *env {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	clock := &game.FakeClock{T: time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)}
	if err := service.New(db, clock).EnsurePlayer(context.Background(), "luk"); err != nil {
		t.Fatal(err)
	}
	return &env{t: t, app: New(db, clock), db: db, clock: clock}
}

func TestFreshInstallIsPlayable(t *testing.T) {
	e := newFreshEnv(t)
	st := e.state()
	if st.Player.Name != "luk" || st.Player.FocusPoints != 0 || len(st.Plots) != 1 || st.Plots[0].State != "empty" {
		t.Fatalf("fresh state = %+v", st)
	}
	var unlocked []string
	for _, s := range st.Seeds {
		if s.Unlocked {
			unlocked = append(unlocked, s.Key)
		}
	}
	if len(st.Seeds) != 5 || len(unlocked) != 1 || unlocked[0] != "daisy" {
		t.Fatalf("seeds = %+v", st.Seeds)
	}
	e.expect(201, "POST", "/api/pomodoros", service.PlantRequest{PlotID: st.Plots[0].ID, PlantType: "daisy"})
}

func TestEnsurePlayerIsIdempotent(t *testing.T) {
	e := newFreshEnv(t)
	dbExec(t, e, `UPDATE players SET focus_points = 7`)
	if err := service.New(e.db, e.clock).EnsurePlayer(context.Background(), "otro"); err != nil {
		t.Fatal(err)
	}
	st := e.state()
	if st.Player.Name != "luk" || st.Player.FocusPoints != 7 || len(st.Plots) != 1 {
		t.Fatalf("second EnsurePlayer changed things: %+v", st)
	}
}

func TestUnlockSeedSpendsFocusOnce(t *testing.T) {
	e := newFreshEnv(t)
	body := map[string]string{"kind": "seed", "key": "tomato"}
	if code := errCode(e.expect(409, "POST", "/api/unlocks", body)); code != "insufficient_focus" {
		t.Fatalf("code = %s", code)
	}
	dbExec(t, e, `UPDATE players SET focus_points = 10`)
	if code := errCode(e.expect(403, "POST", "/api/pomodoros", service.PlantRequest{PlotID: 1, PlantType: "tomato"})); code != "seed_locked" {
		t.Fatalf("planting locked seed: %s", code)
	}
	e.expect(201, "POST", "/api/unlocks", body)
	st := e.state()
	if st.Player.FocusPoints != 2 { // 10 − 8
		t.Fatalf("focus = %d, want 2", st.Player.FocusPoints)
	}
	for _, s := range st.Seeds {
		if s.Key == "tomato" && !s.Unlocked {
			t.Fatal("tomato not unlocked")
		}
	}
	if code := errCode(e.expect(409, "POST", "/api/unlocks", body)); code != "already_unlocked" {
		t.Fatalf("second purchase code = %s", code)
	}
	if e.state().Player.FocusPoints != 2 {
		t.Fatal("second purchase charged again")
	}
	e.expect(201, "POST", "/api/pomodoros", service.PlantRequest{PlotID: 1, PlantType: "tomato"})
	if e.state().Pomodoro.PlannedS != 1500 {
		t.Fatal("tomato should plan 25 min")
	}
}

func TestUnlockValidation(t *testing.T) {
	e := newFreshEnv(t)
	e.expect(400, "POST", "/api/unlocks", map[string]string{"kind": "seed", "key": "potato"})
	e.expect(400, "POST", "/api/unlocks", map[string]string{"kind": "animal", "key": "dog"})
	if code := errCode(e.expect(409, "POST", "/api/unlocks", map[string]string{"kind": "seed", "key": "daisy"})); code != "already_unlocked" {
		t.Fatalf("daisy code = %s", code)
	}
}

// Earning 💧 and spending them end to end: harvest, then buy.
func TestHarvestThenBuyTomatoes(t *testing.T) {
	e := newFreshEnv(t)
	for i := 0; i < 8; i++ { // 8 daisies = 8 💧
		e.expect(201, "POST", "/api/pomodoros", service.PlantRequest{PlotID: 1, PlantType: "daisy"})
		e.clock.Advance(10 * time.Minute)
		e.expect(200, "POST", "/api/plots/1/harvest", nil)
		dbExec(t, e, `UPDATE plots SET state='empty', harvested=0 WHERE id=1`) // stand-in for removing the plant (later task)
	}
	if got := e.state().Player.FocusPoints; got != 8 {
		t.Fatalf("focus after 8 daisies = %d", got)
	}
	e.expect(201, "POST", "/api/unlocks", map[string]string{"kind": "seed", "key": "tomato"})
	if got := e.state().Player.FocusPoints; got != 0 {
		t.Fatalf("focus after buying = %d", got)
	}
}

func TestRecentTagsOrderedByLatestUse(t *testing.T) {
	e := newFreshEnv(t)
	for _, tag := range []string{"emails", "tesis", "emails", "gym"} {
		e.expect(201, "POST", "/api/pomodoros", service.PlantRequest{PlotID: 1, PlantType: "daisy", Tag: tag})
		if tag == "emails" || tag == "tesis" {
			if got := *e.state().Pomodoro.Tag; got != tag {
				t.Fatalf("active tag = %q, want %q", got, tag)
			}
		}
		e.expect(200, "POST", "/api/pomodoros/active/cancel", nil)
	}
	got := e.state().RecentTags
	want := []string{"gym", "emails", "tesis"}
	if len(got) != 3 || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Fatalf("recent tags = %v, want %v", got, want)
	}
	e.expect(201, "POST", "/api/pomodoros", service.PlantRequest{PlotID: 1, PlantType: "daisy"})
	if e.state().Pomodoro.Tag != nil {
		t.Fatal("untagged Pomodoro reports a tag")
	}
}

func TestSettingsAllowlist(t *testing.T) {
	e := newFreshEnv(t)
	e.expect(200, "POST", "/api/settings", map[string]string{"key": "tutorial_done", "value": "1"})
	if e.state().Settings["tutorial_done"] != "1" {
		t.Fatalf("settings = %v", e.state().Settings)
	}
	e.expect(200, "POST", "/api/settings", map[string]string{"key": "tutorial_done", "value": "0"}) // upsert
	if e.state().Settings["tutorial_done"] != "0" {
		t.Fatal("setting not updated")
	}
	e.expect(400, "POST", "/api/settings", map[string]string{"key": "players_drop", "value": "x"})
	e.expect(400, "POST", "/api/settings", map[string]string{"key": "hints_seen", "value": string(make([]byte, 300))})
}

func TestClearPlotRules(t *testing.T) {
	e := newFreshEnv(t)
	code := func(want int, path string, body any) string { return errCode(e.expect(want, "POST", path, body)) }

	if c := code(409, "/api/plots/1/clear", nil); c != "wrong_state" {
		t.Fatalf("clearing an empty plot: %s", c)
	}
	e.expect(201, "POST", "/api/pomodoros", service.PlantRequest{PlotID: 1, PlantType: "daisy"})
	if c := code(409, "/api/plots/1/clear", map[string]bool{"confirm": true}); c != "plot_busy" {
		t.Fatalf("clearing a growing plant: %s", c)
	}
	e.clock.Advance(10 * time.Minute)
	if c := code(409, "/api/plots/1/clear", map[string]bool{"confirm": true}); c != "harvest_first" {
		t.Fatalf("clearing before harvest must protect the reward: %s", c)
	}
	e.expect(200, "POST", "/api/plots/1/harvest", nil)
	if c := code(409, "/api/plots/1/clear", nil); c != "needs_confirmation" {
		t.Fatalf("clearing without confirmation: %s", c)
	}
	if e.state().Plots[0].State != "mature" {
		t.Fatal("plot cleared without confirmation")
	}
	e.expect(200, "POST", "/api/plots/1/clear", map[string]bool{"confirm": true})
	st := e.state()
	if st.Plots[0].State != "empty" || st.Plots[0].PlantType != nil || st.Player.FocusPoints != 1 {
		t.Fatalf("after clear: %+v focus=%d", st.Plots[0], st.Player.FocusPoints)
	}
	e.expect(201, "POST", "/api/pomodoros", service.PlantRequest{PlotID: 1, PlantType: "daisy"}) // replantable
	e.expect(404, "POST", "/api/plots/99/clear", map[string]bool{"confirm": true})
}

// The full single-plot loop the MVP promises: plant, harvest, clear, plant again.
func TestSinglePlotLoopRepeats(t *testing.T) {
	e := newFreshEnv(t)
	for i := 1; i <= 3; i++ {
		e.expect(201, "POST", "/api/pomodoros", service.PlantRequest{PlotID: 1, PlantType: "daisy"})
		e.clock.Advance(10 * time.Minute)
		e.expect(200, "POST", "/api/plots/1/harvest", nil)
		e.expect(200, "POST", "/api/plots/1/clear", map[string]bool{"confirm": true})
		if got := e.state().Player.FocusPoints; got != int64(i) {
			t.Fatalf("loop %d: focus = %d", i, got)
		}
	}
}
