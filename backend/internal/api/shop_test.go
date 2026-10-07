package api

import (
	"sync"
	"testing"

	"github.com/luk224/pomofarm/backend/internal/game"
	"github.com/luk224/pomofarm/backend/internal/service"
)

func giveFocus(t *testing.T, e *env, n int) {
	t.Helper()
	dbExec(t, e, `UPDATE players SET focus_points = ?`, n)
}

func buy(e *env, code int) service.State {
	e.t.Helper()
	e.expect(code, "POST", "/api/plots", nil)
	return e.state()
}

// GDD §4.4: the 15 extra plots cost 3,5,6,9,12,17,23,32,45,62,87,122,171,239,334 💧 = 1.167 in total.
func TestBuyingAllPlotsCostsExactlyWhatTheGDDTableSays(t *testing.T) {
	e := newFreshEnv(t)
	giveFocus(t, e, 1167)
	want := []int{3, 5, 6, 9, 12, 17, 23, 32, 45, 62, 87, 122, 171, 239, 334}
	if got := e.state().Shop.NextPlot; got == nil || got.Cost != 3 || got.Number != 2 {
		t.Fatalf("first offer = %+v", got)
	}
	for i, cost := range want {
		before := e.state().Player.FocusPoints
		offer := e.state().Shop.NextPlot
		if offer == nil || offer.Cost != cost {
			t.Fatalf("plot %d offered at %+v, GDD says %d", i+2, offer, cost)
		}
		st := buy(e, 201)
		if spent := before - st.Player.FocusPoints; spent != int64(cost) {
			t.Fatalf("plot %d cost %d, GDD says %d", i+2, spent, cost)
		}
		if len(st.Plots) != i+2 {
			t.Fatalf("after buying plot %d the farm has %d plots", i+2, len(st.Plots))
		}
	}
	st := e.state()
	if st.Player.FocusPoints != 0 || st.Shop.NextPlot != nil || st.Shop.PlotsOwned != 16 {
		t.Fatalf("after all 15: focus %d, offer %+v, owned %d", st.Player.FocusPoints, st.Shop.NextPlot, st.Shop.PlotsOwned)
	}
	if code := errCode(e.expect(409, "POST", "/api/plots", nil)); code != "maxed_out" {
		t.Fatalf("17th plot: %s", code)
	}
}

func TestNewPlotsAppearWhereTheGridOrderSaysAndNeverOverlap(t *testing.T) {
	e := newFreshEnv(t)
	giveFocus(t, e, 1167)
	cells := map[[2]int]bool{}
	for i := 1; i <= 16; i++ {
		if i > 1 {
			e.expect(201, "POST", "/api/plots", nil)
		}
	}
	for _, p := range e.state().Plots {
		cells[[2]int{p.X, p.Y}] = true
	}
	if len(cells) != 16 {
		t.Fatalf("%d distinct cells, want 16", len(cells))
	}
	for n := 1; n <= 16; n++ {
		x, y, _ := game.PlotPosition(n)
		if !cells[[2]int{x, y}] {
			t.Fatalf("plot %d expected at (%d,%d)", n, x, y)
		}
	}
}

func TestNotEnoughFocusChangesNothing(t *testing.T) {
	e := newFreshEnv(t)
	giveFocus(t, e, 2) // plot 2 costs 3
	if code := errCode(e.expect(409, "POST", "/api/plots", nil)); code != "insufficient_focus" {
		t.Fatalf("code = %s", code)
	}
	st := e.state()
	if st.Player.FocusPoints != 2 || len(st.Plots) != 1 {
		t.Fatalf("a failed purchase changed the game: focus %d, plots %d", st.Player.FocusPoints, len(st.Plots))
	}
}

// GDD §4.5: Silo levels cost 25, 70, 160, 350 💧 (605 total) and hold 24, 36, 48, 72 h.
func TestSiloUpgradesFollowTheGDDTable(t *testing.T) {
	e := newFreshEnv(t)
	giveFocus(t, e, 605)
	if got := e.state().Shop.SiloUpgrade; got == nil || got.Cost != 25 || got.CapacityHours != 24 {
		t.Fatalf("first offer = %+v", got)
	}
	for _, c := range []struct{ cost, hours int }{{25, 24}, {70, 36}, {160, 48}, {350, 72}} {
		before := e.state().Player.FocusPoints
		e.expect(200, "POST", "/api/silo/upgrade", nil)
		st := e.state()
		if before-st.Player.FocusPoints != int64(c.cost) || st.Silo.CapacityHours != c.hours {
			t.Fatalf("upgrade cost %d and capacity %d h, GDD says %d / %d h", before-st.Player.FocusPoints, st.Silo.CapacityHours, c.cost, c.hours)
		}
	}
	if st := e.state(); st.Player.FocusPoints != 0 || st.Shop.SiloUpgrade != nil || st.Player.SiloLevel != 4 {
		t.Fatalf("at max level: focus %d, offer %+v, level %d", st.Player.FocusPoints, st.Shop.SiloUpgrade, st.Player.SiloLevel)
	}
	if code := errCode(e.expect(409, "POST", "/api/silo/upgrade", nil)); code != "maxed_out" {
		t.Fatalf("beyond level 4: %s", code)
	}
}

func TestSiloUpgradeNeedsFocusAndRaisesTheCapOfWhatIsAlreadyStored(t *testing.T) {
	e := newFreshEnv(t)
	plantAndMature(t, e)
	e.clock.Advance(30 * 60 * 60 * 1e9) // 30 h: the 12 h Silo is full (10 🪙 of the 20 the daisy made)
	if st := e.state(); !st.Silo.Full {
		t.Fatal("setup: Silo should be full")
	}
	if code := errCode(e.expect(409, "POST", "/api/silo/upgrade", nil)); code != "insufficient_focus" {
		t.Fatalf("code = %s", code)
	}
	giveFocus(t, e, 25)
	e.expect(200, "POST", "/api/silo/upgrade", nil)
	st := e.state()
	if st.Silo.Full || st.Silo.CapacityHours != 24 || st.Silo.CapacityMilli < 19900 {
		t.Fatalf("after upgrading: %+v (24 h of 0.83 🪙/h should hold ≈20)", st.Silo)
	}
}

func TestPurchasesDoNotInterruptProductionOrAPomodoro(t *testing.T) {
	e := newFreshEnv(t)
	plantAndMature(t, e)
	e.clock.Advance(5 * 60 * 60 * 1e9)
	giveFocus(t, e, 30)
	before, _ := milli(t, e)
	buy(e, 201)
	e.expect(200, "POST", "/api/silo/upgrade", nil)
	after, _ := milli(t, e)
	near(t, "Silo content unchanged by purchases", after, before, 2)
	// and a new plot can grow a Pomodoro while the first one keeps producing
	e.expect(201, "POST", "/api/pomodoros", service.PlantRequest{PlotID: 2, PlantType: "daisy"})
	if e.state().Pomodoro == nil || e.state().Silo.RateMilliPerHour == 0 {
		t.Fatal("planting in the new plot stopped the first plant")
	}
}

func TestConcurrentPurchasesNeverSpendTwice(t *testing.T) {
	e := newFreshEnv(t)
	giveFocus(t, e, 3) // exactly one plot
	var wg sync.WaitGroup
	codes := make([]int, 6)
	for i := range codes {
		wg.Add(1)
		go func(i int) { defer wg.Done(); codes[i], _ = e.do("POST", "/api/plots", nil) }(i)
	}
	wg.Wait()
	created := 0
	for _, c := range codes {
		if c == 201 {
			created++
		}
	}
	st := e.state()
	if created != 1 || len(st.Plots) != 2 || st.Player.FocusPoints != 0 {
		t.Fatalf("codes %v: plots %d, focus %d (3 💧 buy exactly one plot)", codes, len(st.Plots), st.Player.FocusPoints)
	}
}
