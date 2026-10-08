package api

import (
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/luk224/pomofarm/backend/internal/game"
	"github.com/luk224/pomofarm/backend/internal/service"
)

// readyFarm builds the farm the GDD asks for before a new season: 16 plots, Silo level 4, four hives, the Dog.
// Plants are not part of the requirement, so none are planted here.
func readyFarm(t *testing.T, e *env) map[[2]int]int64 {
	t.Helper()
	var cells map[[2]int]int64
	if len(e.state().Plots) < 16 {
		cells = fullFarm(t, e)
	} else { // plots survive a prestige: only the automation has to be rebuilt
		cells = map[[2]int]int64{}
		for _, p := range e.state().Plots {
			cells[[2]int{p.X, p.Y}] = p.ID
		}
	}
	unlockAnimals(t, e)
	dbExec(t, e, `UPDATE players SET silo_level = 4, coins_milli = 100000000`)
	for _, c := range [][2]int{{1, 1}, {2, 1}, {1, 2}, {2, 2}} {
		buyHive(e, 201, cells[c])
	}
	e.expect(201, "POST", "/api/structures", map[string]any{"kind": "dog"})
	return cells
}

func prestige(e *env, code int, confirm bool) []byte {
	e.t.Helper()
	return e.expect(code, "POST", "/api/prestige", map[string]bool{"confirm": confirm})
}

func TestPrestigeRequirementsAreListedAndAllNeeded(t *testing.T) {
	e := newFreshEnv(t)
	p := e.state().Prestige
	if p.Ready || len(p.Requirements) != 4 || p.Season != 1 || p.NextSeason != 2 || p.NextBiome != "summer" || p.BonusPct != 0 || p.NextBonusPct != 10 {
		t.Fatalf("a new game: %+v", p)
	}
	want := map[string][2]int{"plots": {1, 16}, "silo": {0, 4}, "hives": {0, 4}, "dog": {0, 1}}
	for _, r := range p.Requirements {
		if w, ok := want[r.Key]; !ok || r.Have != w[0] || r.Need != w[1] || r.Met {
			t.Fatalf("requirement %+v", r)
		}
	}
	prestige(e, 409, true) // not ready: refused even when confirmed
}

func TestEachMissingRequirementBlocksThePrestige(t *testing.T) {
	for _, missing := range []string{"plots", "silo", "hives", "dog"} {
		e := newFreshEnv(t)
		cells := readyFarm(t, e)
		switch missing {
		case "plots":
			dbExec(t, e, `DELETE FROM structures WHERE kind = 'hive' AND x = 3`) // keep hives; remove a plot with none on it
			dbExec(t, e, `DELETE FROM plots WHERE x = 3 AND y = 3`)
		case "silo":
			dbExec(t, e, `UPDATE players SET silo_level = 3`)
		case "hives":
			dbExec(t, e, `DELETE FROM structures WHERE id = (SELECT MAX(id) FROM structures WHERE kind = 'hive')`)
		case "dog":
			dbExec(t, e, `DELETE FROM structures WHERE kind = 'dog'`)
		}
		_ = cells
		before := e.state()
		if before.Prestige.Ready {
			t.Fatalf("without %s the farm must not be ready: %+v", missing, before.Prestige)
		}
		prestige(e, 409, true)
		after := e.state()
		if after.Player.Season != 1 || len(after.Plots) != len(before.Plots) || after.Player.CoinsMilli != before.Player.CoinsMilli {
			t.Fatalf("a refused prestige must change nothing (missing %s)", missing)
		}
	}
}

func TestPrestigeNeedsConfirmation(t *testing.T) {
	e := newFreshEnv(t)
	readyFarm(t, e)
	if !e.state().Prestige.Ready {
		t.Fatal("the farm should be ready")
	}
	prestige(e, 409, false)
	e.expect(409, "POST", "/api/prestige", nil)
	if e.state().Player.Season != 1 {
		t.Fatal("no season without confirmation")
	}
}

// What the GDD says starts over, and what stays (§4.9).
func TestPrestigeResetsExactlyWhatTheGDDSays(t *testing.T) {
	e := newFreshEnv(t)
	cells := readyFarm(t, e)
	giveFocus(t, e, 400)
	// a decorated farm with a hat, mature plants, coins in the balance and in the Silo, an unlocked seed, history
	e.expect(201, "POST", "/api/decor", map[string]any{"kind": "lantern", "x": 4, "y": 1})
	e.expect(201, "POST", "/api/decor", map[string]any{"kind": "hat"})
	e.expect(201, "POST", "/api/unlocks", map[string]string{"kind": "seed", "key": "tomato"})
	e.expect(201, "POST", "/api/pomodoros", service.PlantRequest{PlotID: cells[[2]int{0, 0}], PlantType: "daisy", Tag: "tesis"})
	e.clock.Advance(11 * time.Minute)
	e.expect(200, "POST", "/api/plots/"+itoa(cells[[2]int{0, 0}])+"/harvest", nil)
	e.expect(201, "POST", "/api/pomodoros", service.PlantRequest{PlotID: cells[[2]int{3, 3}], PlantType: "tomato"})
	e.clock.Advance(30 * time.Minute) // mature, NOT harvested
	e.clock.Advance(2 * time.Hour)    // and producing, so the Silo holds some
	before := e.state()
	// (the Dog empties the Silo by himself, so what the farm made is in the balance)
	if before.Player.CoinsMilli <= 0 {
		t.Fatalf("test setup: coins %d", before.Player.CoinsMilli)
	}
	var pomodoros int
	e.db.QueryRow(`SELECT COUNT(*) FROM pomodoros`).Scan(&pomodoros)
	tomato, _ := game.CropByKey("tomato")
	tomatoReward := tomato.Reward

	prestige(e, 200, true)
	st := e.state()
	if st.Player.Season != 2 || st.Player.Biome != "summer" {
		t.Fatalf("season %d biome %q", st.Player.Season, st.Player.Biome)
	}
	// reset: plants, structures, 🪙, Silo content
	for _, p := range st.Plots {
		if p.State != "empty" || p.PlantType != nil || p.Harvested || p.Bonus != nil {
			t.Fatalf("plot %d should be empty: %+v", p.ID, p)
		}
	}
	if len(st.Automation.Bees.Hives) != 0 || st.Automation.Dog.Owned || len(st.Decor.Items) != 0 || st.Decor.Hat.Owned {
		t.Fatalf("structures must start over: %+v %+v", st.Automation, st.Decor)
	}
	if st.Player.CoinsMilli != 0 || st.Silo.ContentMilli != 0 {
		t.Fatalf("🪙 must start over: coins %d silo %d", st.Player.CoinsMilli, st.Silo.ContentMilli)
	}
	// kept: 💧 (and the unharvested tomato was harvested for the player first), seeds, animal unlocks, plots, Silo level, history
	if st.Player.FocusPoints != before.Player.FocusPoints+int64(tomatoReward) {
		t.Fatalf("💧 %d, want %d + the tomato's %d", st.Player.FocusPoints, before.Player.FocusPoints, tomatoReward)
	}
	if st.Player.LifetimeFocus != before.Player.LifetimeFocus+int64(tomatoReward) {
		t.Fatalf("lifetime 💧 %d", st.Player.LifetimeFocus)
	}
	if len(st.Plots) != 16 || st.Player.SiloLevel != 4 {
		t.Fatalf("plots %d, Silo level %d", len(st.Plots), st.Player.SiloLevel)
	}
	unlocked := 0
	for _, s := range st.Seeds {
		if s.Unlocked {
			unlocked++
		}
	}
	if unlocked != 2 || !st.Automation.Bees.Unlocked || !st.Automation.Dog.Unlocked {
		t.Fatalf("unlocks kept: %d seeds, bees %v, dog %v", unlocked, st.Automation.Bees.Unlocked, st.Automation.Dog.Unlocked)
	}
	var after int
	e.db.QueryRow(`SELECT COUNT(*) FROM pomodoros`).Scan(&after)
	if after != pomodoros {
		t.Fatalf("the Harvest Book must keep every Pomodoro: %d -> %d", pomodoros, after)
	}
	if code, b := getBook(t, e, "?tz=UTC"); code != 200 || b.Pomodoros != 2 {
		t.Fatalf("book after prestige: %+v", b)
	}
	if p := st.Prestige; p.Season != 2 || p.BonusPct != 10 || p.NextBonusPct != 20 || p.NextBiome != "autumn" || p.Ready {
		t.Fatalf("prestige state after: %+v", p)
	}
}

func TestPrestigeRefusesWhileAPomodoroRuns(t *testing.T) {
	e := newFreshEnv(t)
	cells := readyFarm(t, e)
	e.expect(201, "POST", "/api/pomodoros", service.PlantRequest{PlotID: cells[[2]int{0, 0}], PlantType: "daisy"})
	prestige(e, 409, true)
	if st := e.state(); st.Player.Season != 1 || st.Pomodoro == nil {
		t.Fatal("the running Pomodoro and the season must be untouched")
	}
}

func TestPrestigeEndsAPendingRest(t *testing.T) {
	e := newFreshEnv(t)
	cells := readyFarm(t, e)
	e.expect(201, "POST", "/api/pomodoros", service.PlantRequest{PlotID: cells[[2]int{0, 0}], PlantType: "daisy"})
	e.clock.Advance(11 * time.Minute)
	e.expect(200, "POST", "/api/plots/"+itoa(cells[[2]int{0, 0}])+"/harvest", nil)
	if e.state().Rest == nil {
		t.Fatal("test setup: harvesting should offer a rest")
	}
	prestige(e, 200, true)
	if e.state().Rest != nil {
		t.Fatal("a rest from the old season must not carry over")
	}
}

func TestTheNewSeasonEarnsTenPercentMore(t *testing.T) {
	rate := func(season int) int64 {
		e := newFreshEnv(t)
		if season > 1 {
			readyFarm(t, e)
			for i := 1; i < season; i++ {
				prestige(e, 200, true)
				if i < season-1 {
					readyFarm(t, e)
				}
			}
		}
		e.expect(201, "POST", "/api/pomodoros", service.PlantRequest{PlotID: 1, PlantType: "daisy"})
		e.clock.Advance(11 * time.Minute)
		return e.state().Silo.RateMilliPerHour
	}
	base := rate(1)
	if base <= 0 {
		t.Fatal("no production in season 1")
	}
	for season := 2; season <= 3; season++ {
		got := rate(season)
		want := float64(base) * (1 + 0.1*float64(season-1))
		if d := float64(got) - want; d > 2 || d < -2 {
			t.Fatalf("season %d earns %d milli/h, want %.0f (+%d%%)", season, got, want, (season-1)*10)
		}
	}
}

func TestBiomesFollowTheSeasons(t *testing.T) {
	for season, want := range map[int]string{1: "spring", 2: "summer", 3: "autumn", 4: "winter", 5: "spring", 6: "summer", 0: "spring", -3: "spring"} {
		if got := game.BiomeForSeason(season); got != want {
			t.Errorf("season %d: %q, want %q", season, got, want)
		}
	}
}

func TestOnlyOnePrestigeWinsARace(t *testing.T) {
	e := newFreshEnv(t)
	readyFarm(t, e)
	var wg sync.WaitGroup
	codes := make([]int, 6)
	for i := range codes {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			codes[i], _ = e.do("POST", "/api/prestige", map[string]bool{"confirm": true})
		}(i)
	}
	wg.Wait()
	ok := 0
	for _, c := range codes {
		if c == 200 {
			ok++
		}
	}
	if st := e.state(); ok != 1 || st.Player.Season != 2 {
		t.Fatalf("codes %v, season %d", codes, st.Player.Season)
	}
}

func TestAfterAPrestigeTheAutomationCanBeBuiltAgain(t *testing.T) {
	e := newFreshEnv(t)
	cells := readyFarm(t, e)
	prestige(e, 200, true)
	giveCoins(t, e, 100000)
	buyHive(e, 201, cells[[2]int{1, 1}])
	if a := e.state().Automation.Bees; len(a.Hives) != 1 || a.NextCost == nil || *a.NextCost != 6000 {
		t.Fatalf("hive prices start over too: %+v", a)
	}
	e.expect(201, "POST", "/api/structures", map[string]any{"kind": "dog"})
	e.expect(201, "POST", "/api/decor", map[string]any{"kind": "path", "x": 4, "y": 1})
}

func TestPrestigeOnAGameOlderThanThePrestigeColumns(t *testing.T) {
	e := newFreshEnv(t)
	dbExec(t, e, `UPDATE players SET biome = 'spring', season = 1`)
	if b := e.state().Player.Biome; b != "spring" {
		t.Fatalf("an existing save reads as spring, got %q", b)
	}
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }
