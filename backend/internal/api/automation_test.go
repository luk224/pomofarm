package api

import (
	"fmt"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/luk224/pomofarm/backend/internal/game"
)

func giveCoins(t *testing.T, e *env, coins int64) {
	t.Helper()
	dbExec(t, e, `UPDATE players SET coins_milli = ?`, coins*1000)
}

func unlockAnimals(t *testing.T, e *env) {
	t.Helper()
	dbExec(t, e, `INSERT OR IGNORE INTO unlocks (player_id,kind,key,at) VALUES (1,'animal','bees','t'),(1,'animal','dog','t')`)
}

func buyHive(e *env, code int, plotID int64) []byte {
	e.t.Helper()
	return e.expect(code, "POST", "/api/structures", map[string]any{"kind": "hive", "plot_id": plotID})
}

func coinsOf(e *env) int64 { return e.state().Player.CoinsMilli }

// fullFarm buys all 16 plots and returns them keyed by grid cell.
func fullFarm(t *testing.T, e *env) map[[2]int]int64 {
	t.Helper()
	giveFocus(t, e, 3000)
	for i := 0; i < 15; i++ {
		e.expect(201, "POST", "/api/plots", nil)
	}
	cells := map[[2]int]int64{}
	for _, p := range e.state().Plots {
		cells[[2]int{p.X, p.Y}] = p.ID
	}
	return cells
}

func TestAnimalsUnlockWithFocusAtTheGDDPrices(t *testing.T) {
	e := newFreshEnv(t)
	giveFocus(t, e, 29)
	if code := errCode(e.expect(409, "POST", "/api/unlocks", map[string]string{"kind": "animal", "key": "bees"})); code != "insufficient_focus" {
		t.Fatalf("29 💧 for the bees: %s", code)
	}
	giveFocus(t, e, 30)
	e.expect(201, "POST", "/api/unlocks", map[string]string{"kind": "animal", "key": "bees"})
	if e.state().Player.FocusPoints != 0 || !e.state().Automation.Bees.Unlocked {
		t.Fatalf("after the bees: %+v", e.state().Automation.Bees)
	}
	if code := errCode(e.expect(409, "POST", "/api/unlocks", map[string]string{"kind": "animal", "key": "bees"})); code != "already_unlocked" {
		t.Fatalf("second purchase: %s", code)
	}
	giveFocus(t, e, 120)
	e.expect(201, "POST", "/api/unlocks", map[string]string{"kind": "animal", "key": "dog"})
	e.expect(400, "POST", "/api/unlocks", map[string]string{"kind": "animal", "key": "cat"})
}

// GDD §4.7: hives cost 4.000 / 6.000 / 9.000 / 13.500 🪙, the Dog 30.000: 62.500 🪙 in total.
func TestAutomationCostsAddUpToTheGDDTotal(t *testing.T) {
	e := newFreshEnv(t)
	cells := fullFarm(t, e)
	unlockAnimals(t, e)
	giveCoins(t, e, 62500)
	centre := []int64{cells[[2]int{1, 1}], cells[[2]int{2, 1}], cells[[2]int{1, 2}], cells[[2]int{2, 2}]}
	if a := e.state().Automation.Bees; a.NextCost == nil || *a.NextCost != 4000 {
		t.Fatalf("first offer: %+v", a)
	}
	for i, want := range []int64{4000, 6000, 9000, 13500} {
		before := coinsOf(e)
		buyHive(e, 201, centre[i])
		if spent := (before - coinsOf(e)) / 1000; spent != want {
			t.Fatalf("hive %d cost %d 🪙, GDD %d", i+1, spent, want)
		}
	}
	if a := e.state().Automation.Bees; a.NextCost != nil || len(a.Hives) != 4 {
		t.Fatalf("after four hives: %+v", a)
	}
	before := coinsOf(e)
	e.expect(201, "POST", "/api/structures", map[string]any{"kind": "dog"})
	if (before-coinsOf(e))/1000 != 30000 {
		t.Fatalf("dog cost %d", (before-coinsOf(e))/1000)
	}
	if coinsOf(e) != 0 {
		t.Fatalf("62.500 🪙 should buy everything exactly, %d left", coinsOf(e))
	}
	if code := errCode(buyHive(e, 409, cells[[2]int{0, 0}])); code != "maxed_out" {
		t.Fatalf("fifth hive: %s", code)
	}
	giveCoins(t, e, 40000)
	if code := errCode(e.expect(409, "POST", "/api/structures", map[string]any{"kind": "dog"})); code != "already_owned" {
		t.Fatalf("second dog: %s", code)
	}
}

func TestStructuresNeedTheirUnlockAndEnoughCoins(t *testing.T) {
	e := newFreshEnv(t)
	giveCoins(t, e, 100000)
	if code := errCode(buyHive(e, 403, 1)); code != "animal_locked" {
		t.Fatalf("hive before unlocking the bees: %s", code)
	}
	if code := errCode(e.expect(403, "POST", "/api/structures", map[string]any{"kind": "dog"})); code != "animal_locked" {
		t.Fatalf("dog before unlocking: %s", code)
	}
	unlockAnimals(t, e)
	giveCoins(t, e, 3999)
	if code := errCode(buyHive(e, 409, 1)); code != "insufficient_coins" {
		t.Fatalf("3.999 🪙: %s", code)
	}
	st := e.state()
	if st.Player.CoinsMilli != 3999000 || len(st.Automation.Bees.Hives) != 0 {
		t.Fatalf("a failed purchase changed the game: %d milli, %d hives", st.Player.CoinsMilli, len(st.Automation.Bees.Hives))
	}
	e.expect(400, "POST", "/api/structures", map[string]any{"kind": "windmill"})
	if code := errCode(buyHive(e, 404, 999)); code != "not_found" {
		t.Fatalf("hive beside a plot that does not exist: %s", code)
	}
}

// GDD §4.6: bees give +25% to every plot in the 3×3 area around the hive.
func TestAHiveBoostsItsThreeByThreeArea(t *testing.T) {
	e := newFreshEnv(t)
	cells := fullFarm(t, e)
	unlockAnimals(t, e)
	giveCoins(t, e, 100000)
	dbExec(t, e, `UPDATE players SET silo_level = 4`)
	for _, id := range cells { // all daisies: no adjacency or garden, so only bees can add a bonus
		mature(t, e, int(id), "daisy", 0, 24)
	}
	buyHive(e, 201, cells[[2]int{1, 1}])
	e.clock.Advance(time.Hour)
	st := e.state()
	covered := 0
	for _, p := range st.Plots {
		inArea := p.X >= 0 && p.X <= 2 && p.Y >= 0 && p.Y <= 2
		if p.Bonus == nil {
			t.Fatalf("plot %d has no bonus block", p.ID)
		}
		want := 1.0
		if inArea {
			want = 1.25
			covered++
		}
		if math.Abs(p.Bonus.Multiplier-want) > 1e-9 || p.Bonus.Bees != inArea {
			t.Errorf("plot (%d,%d): %+v, want ×%.2f bees=%v", p.X, p.Y, p.Bonus, want, inArea)
		}
	}
	if covered != 9 {
		t.Fatalf("%d plots covered, want 9", covered)
	}
}

// "No se acumulan entre colmenas": four hives on the central 2×2 cover all 16 plots, each at +25%, not more.
func TestFourHivesCoverTheWholeFieldWithoutStacking(t *testing.T) {
	e := newFreshEnv(t)
	cells := fullFarm(t, e)
	unlockAnimals(t, e)
	giveCoins(t, e, 100000)
	for _, id := range cells {
		mature(t, e, int(id), "daisy", 0, 24)
	}
	for _, c := range [][2]int{{1, 1}, {2, 1}, {1, 2}, {2, 2}} {
		buyHive(e, 201, cells[c])
	}
	e.clock.Advance(time.Hour)
	for _, p := range e.state().Plots {
		if p.Bonus == nil || math.Abs(p.Bonus.Multiplier-1.25) > 1e-9 || !p.Bonus.Bees {
			t.Fatalf("plot (%d,%d): %+v, want exactly ×1.25 from the hives", p.X, p.Y, p.Bonus)
		}
	}
}

func TestBeesOnlyHelpPlantsThatAreProducing(t *testing.T) {
	e := newFreshEnv(t)
	plotsFor(t, e, 4)
	unlockAnimals(t, e)
	giveCoins(t, e, 10000)
	mature(t, e, 1, "daisy", 0, 24) // producing
	mature(t, e, 2, "daisy", 0, 24) // will wilt
	dbExec(t, e, `UPDATE plots SET wilts_at = ? WHERE id = 2`, e.clock.T.Add(30*time.Minute).Format(time.RFC3339Nano))
	buyHive(e, 201, 1)
	e.clock.Advance(time.Hour)
	st := e.state()
	if st.Plots[0].Bonus == nil && st.Plots[1].Bonus == nil {
		t.Fatal("no plant has a bonus")
	}
	for _, p := range st.Plots {
		switch {
		case p.State == "withered" && p.Bonus != nil, p.State == "empty" && p.Bonus != nil:
			t.Fatalf("plot %d (%s) got a bee bonus: %+v", p.ID, p.State, p.Bonus)
		}
	}
}

// The purchase settles production first: before it the plants earn no bonus, from it on they do.
func TestHiveBoughtMidWindowOnlyBoostsFromThatMoment(t *testing.T) {
	e := newFreshEnv(t)
	unlockAnimals(t, e)
	giveCoins(t, e, 4000)
	dbExec(t, e, `UPDATE players SET silo_level = 4`)
	mature(t, e, 1, "daisy", 0, 24)
	e.clock.Advance(5 * time.Hour)
	buyHive(e, 201, 1)
	e.clock.Advance(5 * time.Hour)
	want := baseRatePerHour("daisy", 24) * (5 + 5*1.25) * 1000
	near(t, "Silo: 5 h plain + 5 h with bees", e.state().Silo.ContentMilli, int64(want), 3)
}

func TestMovingAHiveIsFreeAndInstant(t *testing.T) {
	e := newFreshEnv(t)
	cells := fullFarm(t, e)
	unlockAnimals(t, e)
	giveCoins(t, e, 4000)
	dbExec(t, e, `UPDATE players SET silo_level = 4`)
	a, b := cells[[2]int{1, 1}], cells[[2]int{3, 3}] // far apart: no shared area
	mature(t, e, int(a), "daisy", 0, 24)
	mature(t, e, int(b), "daisy", 0, 24)
	buyHive(e, 201, a)
	hive := e.state().Automation.Bees.Hives[0]
	if hive.PlotID != a {
		t.Fatalf("hive beside plot %d, want %d", hive.PlotID, a)
	}
	e.clock.Advance(4 * time.Hour)
	coins := coinsOf(e)
	e.expect(200, "POST", fmt.Sprintf("/api/structures/%d/move", hive.ID), map[string]any{"plot_id": b})
	if coinsOf(e) != coins {
		t.Fatalf("moving cost %d milli: it must be free", coins-coinsOf(e))
	}
	e.clock.Advance(4 * time.Hour)
	st := e.state()
	d := baseRatePerHour("daisy", 24)
	want := d*(4*1.25+4)*1000 + d*(4+4*1.25)*1000 // plot a: bonus the first 4 h; plot b: bonus the last 4 h
	near(t, "Silo after moving the hive", st.Silo.ContentMilli, int64(want), 4)
	for _, p := range st.Plots {
		if p.ID == a && p.Bonus.Multiplier != 1 || p.ID == b && math.Abs(p.Bonus.Multiplier-1.25) > 1e-9 {
			t.Fatalf("plot %d after the move: %+v", p.ID, p.Bonus)
		}
	}
	if st.Automation.Bees.Hives[0].PlotID != b {
		t.Fatalf("hive now beside plot %d, want %d", st.Automation.Bees.Hives[0].PlotID, b)
	}
}

func TestMovingAHiveHasItsRules(t *testing.T) {
	e := newFreshEnv(t)
	cells := fullFarm(t, e)
	unlockAnimals(t, e)
	giveCoins(t, e, 20000)
	buyHive(e, 201, cells[[2]int{1, 1}])
	buyHive(e, 201, cells[[2]int{2, 1}])
	hives := e.state().Automation.Bees.Hives
	path := func(id int64) string { return fmt.Sprintf("/api/structures/%d/move", id) }
	if code := errCode(e.expect(409, "POST", path(hives[0].ID), map[string]any{"plot_id": cells[[2]int{2, 1}]})); code != "cell_taken" {
		t.Fatalf("onto another hive: %s", code)
	}
	if code := errCode(e.expect(409, "POST", path(hives[0].ID), map[string]any{"plot_id": cells[[2]int{1, 1}]})); code != "wrong_state" {
		t.Fatalf("onto its own plot: %s", code)
	}
	e.expect(404, "POST", path(hives[0].ID), map[string]any{"plot_id": 999})
	e.expect(404, "POST", path(999), map[string]any{"plot_id": cells[[2]int{0, 0}]})
	e.expect(400, "POST", "/api/structures/abc/move", map[string]any{"plot_id": 1})
	if code := errCode(buyHive(e, 409, cells[[2]int{1, 1}])); code != "cell_taken" {
		t.Fatalf("second hive on the same plot: %s", code)
	}
}

func TestTheDogEmptiesTheSiloAndAddsTwelveHours(t *testing.T) {
	e := newFreshEnv(t)
	unlockAnimals(t, e)
	giveCoins(t, e, 30000)
	plantAndMature(t, e)
	e.clock.Advance(5 * time.Hour)
	if e.state().Silo.ContentMilli == 0 {
		t.Fatal("setup: the Silo should hold something before the dog")
	}
	e.expect(201, "POST", "/api/structures", map[string]any{"kind": "dog"})
	st := e.state()
	if !st.Automation.Dog.Owned || st.Silo.ContentMilli > 1 || st.Silo.CapacityHours != 24 {
		t.Fatalf("with the dog: owned=%v silo=%d cap=%dh", st.Automation.Dog.Owned, st.Silo.ContentMilli, st.Silo.CapacityHours)
	}
	if st.Player.CoinsMilli < 4000 { // the 5 h of production were collected for the player right away
		t.Fatalf("coins after the dog collected: %d", st.Player.CoinsMilli)
	}
}

func TestConcurrentPurchasesNeverSpendCoinsTwice(t *testing.T) {
	e := newFreshEnv(t)
	cells := fullFarm(t, e)
	unlockAnimals(t, e)
	giveCoins(t, e, 4000) // exactly one hive
	var wg sync.WaitGroup
	codes := make([]int, 6)
	ids := []int64{cells[[2]int{0, 0}], cells[[2]int{1, 0}], cells[[2]int{2, 0}], cells[[2]int{3, 0}], cells[[2]int{0, 1}], cells[[2]int{1, 1}]}
	for i := range codes {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			codes[i], _ = e.do("POST", "/api/structures", map[string]any{"kind": "hive", "plot_id": ids[i]})
		}(i)
	}
	wg.Wait()
	created := 0
	for _, c := range codes {
		if c == 201 {
			created++
		}
	}
	st := e.state()
	if created != 1 || len(st.Automation.Bees.Hives) != 1 || st.Player.CoinsMilli != 0 {
		t.Fatalf("codes %v: %d hives, %d milli left", codes, len(st.Automation.Bees.Hives), st.Player.CoinsMilli)
	}
}

func TestStructureRulesHoldAtTheDatabaseLevelToo(t *testing.T) {
	e := newFreshEnv(t)
	dbExec(t, e, `INSERT INTO structures (player_id,kind,x,y) VALUES (1,'hive',1,1)`)
	for _, q := range []string{
		`INSERT INTO structures (player_id,kind,x,y) VALUES (1,'hive',1,1)`,
		`INSERT INTO structures (player_id,kind,x,y) VALUES (1,'dog',0,0), (1,'dog',3,3)`,
	} {
		if _, err := e.db.Exec(q); err == nil {
			t.Fatalf("the database accepted: %s", q)
		}
	}
	if _, err := e.db.Exec(`INSERT INTO structures (player_id,kind,x,y) VALUES (1,'path',1,1)`); err != nil { // other kinds may share a cell
		t.Fatalf("a path on a hive cell was refused: %v", err)
	}
}

func TestPrestigeAndBeesCombine(t *testing.T) {
	e := newFreshEnv(t)
	unlockAnimals(t, e)
	giveCoins(t, e, 4000)
	dbExec(t, e, `UPDATE players SET season = 3`) // ×1.2 above the cap
	mature(t, e, 1, "daisy", 0, 24)
	buyHive(e, 201, 1)
	e.clock.Advance(time.Hour)
	if b := e.state().Plots[0].Bonus; b == nil || math.Abs(b.Multiplier-1.25*1.2) > 1e-9 {
		t.Fatalf("bonus %+v, want ×%.2f", b, 1.25*1.2)
	}
}

func TestAutomationStateOffersWhatCanBeBought(t *testing.T) {
	e := newFreshEnv(t)
	a := e.state().Automation
	if a.Bees.Unlocked || a.Bees.UnlockCost != 30 || a.Bees.Max != 4 || a.Dog.Unlocked || a.Dog.UnlockCost != 120 || a.Dog.Cost != 30000 || a.Dog.SiloBonusHours != 12 {
		t.Fatalf("fresh automation state: %+v", a)
	}
	if a.Bees.NextCost == nil || *a.Bees.NextCost != game.HiveCost(0) {
		t.Fatalf("next hive: %v", a.Bees.NextCost)
	}
}
