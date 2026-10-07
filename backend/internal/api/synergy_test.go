package api

import (
	"math"
	"testing"
	"time"

	"github.com/luk224/pomofarm/backend/internal/game"
	"github.com/luk224/pomofarm/backend/internal/service"
)

// plotsFor makes sure the farm has n plots (positions follow the grid order of the GDD).
func plotsFor(t *testing.T, e *env, n int) {
	t.Helper()
	for i := 2; i <= n; i++ {
		x, y, _ := game.PlotPosition(i)
		dbExec(t, e, `INSERT INTO plots (player_id, x, y) VALUES (1, ?, ?)`, x, y)
	}
}

// mature puts a mature, harvested plant on plot `id` that matured `maturedAgo` ago and lives `lifeH` hours.
func mature(t *testing.T, e *env, id int, crop string, maturedAgo time.Duration, lifeH int) {
	t.Helper()
	c, _ := game.CropByKey(crop)
	at := e.clock.T.Add(-maturedAgo)
	dbExec(t, e, `UPDATE plots SET state='mature', plant_type=?, grow_s=?, life_s=?, harvested=1, matured_at=?, wilts_at=?, collected_to=? WHERE id=?`,
		crop, c.DurationMin*60, lifeH*3600, at.Format(time.RFC3339Nano), at.Add(time.Duration(lifeH)*time.Hour).Format(time.RFC3339Nano), at.Format(time.RFC3339Nano), id)
}

func baseRatePerHour(crop string, lifeH int) float64 {
	c, _ := game.CropByKey(crop)
	return game.CycleYield(float64(c.DurationMin)) / float64(lifeH)
}

func bonusOf(t *testing.T, st service.State, id int64) service.PlotBonus {
	t.Helper()
	for _, p := range st.Plots {
		if p.ID == id {
			if p.Bonus == nil {
				t.Fatalf("plot %d has no bonus block: it should be producing", id)
			}
			return *p.Bonus
		}
	}
	t.Fatalf("no plot %d", id)
	return service.PlotBonus{}
}

func TestCompatibleNeighboursProduceMore(t *testing.T) {
	e := newFreshEnv(t)
	plotsFor(t, e, 2)                                 // plots at (1,1) and (2,1): side by side
	dbExec(t, e, `UPDATE players SET silo_level = 4`) // 72 h of room: no cap involved
	mature(t, e, 1, "daisy", 0, 24)
	mature(t, e, 2, "tomato", 0, 36)
	e.clock.Advance(10 * time.Hour)
	st := e.state()
	want := (baseRatePerHour("daisy", 24) + baseRatePerHour("tomato", 36)) * 10 * 1.1 * 1000
	near(t, "Silo after 10 h with the daisy–tomato bonus", st.Silo.ContentMilli, int64(want), 3)
	for _, id := range []int64{1, 2} {
		if b := bonusOf(t, st, id); math.Abs(b.Multiplier-1.1) > 1e-9 || b.Neighbours != 1 || b.Garden {
			t.Fatalf("plot %d bonus = %+v, want ×1.1 with one neighbour", id, b)
		}
	}
}

func TestIncompatibleOrSameCropNeighboursEarnNothing(t *testing.T) {
	e := newFreshEnv(t)
	plotsFor(t, e, 3)
	mature(t, e, 1, "daisy", 0, 24)
	mature(t, e, 2, "daisy", 0, 24)     // same crop next to it
	mature(t, e, 3, "sunflower", 0, 54) // (1,2): below plot 1, daisy–sunflower are not ring neighbours
	e.clock.Advance(time.Hour)
	st := e.state()
	for _, id := range []int64{1, 2, 3} {
		if b := bonusOf(t, st, id); b.Multiplier != 1 || b.Neighbours != 0 {
			t.Errorf("plot %d: %+v, want no bonus", id, b)
		}
	}
}

// GDD §4.6, Huerto completo: the central 2×2 with four different crops.
func TestFullGardenInTheCentralBlock(t *testing.T) {
	e := newFreshEnv(t)
	plotsFor(t, e, 4) // (1,1) (2,1) (1,2) (2,2)
	dbExec(t, e, `UPDATE players SET silo_level = 4`)
	for id, crop := range map[int]string{1: "daisy", 2: "tomato", 3: "oak", 4: "sunflower"} {
		c, _ := game.CropByKey(crop)
		mature(t, e, id, crop, 0, c.LifeH)
	}
	e.clock.Advance(5 * time.Hour)
	st := e.state()
	want := map[int64]float64{1: 1.35, 2: 1.35, 3: 1.25, 4: 1.25} // 1 + 10% per compatible neighbour + 15% garden
	var expected float64
	for id, m := range want {
		b := bonusOf(t, st, id)
		if math.Abs(b.Multiplier-m) > 1e-9 || !b.Garden {
			t.Errorf("plot %d: %+v, want ×%.2f with garden", id, b, m)
		}
		crop := map[int64]string{1: "daisy", 2: "tomato", 3: "oak", 4: "sunflower"}[id]
		c, _ := game.CropByKey(crop)
		expected += baseRatePerHour(crop, c.LifeH) * m * 5 * 1000
	}
	near(t, "Silo after 5 h of a full garden", st.Silo.ContentMilli, int64(expected), 4)
}

func TestTheBonusEndsWhenTheNeighbourWilts(t *testing.T) {
	e := newFreshEnv(t)
	plotsFor(t, e, 2)
	dbExec(t, e, `UPDATE players SET silo_level = 4`)
	mature(t, e, 1, "daisy", 0, 24)
	mature(t, e, 2, "tomato", 0, 6) // wilts after 6 h
	e.clock.Advance(10 * time.Hour)
	st := e.state()
	want := baseRatePerHour("daisy", 24)*(6*1.1+4) + baseRatePerHour("tomato", 6)*6*1.1
	near(t, "Silo", st.Silo.ContentMilli, int64(want*1000), 4)
	if b := bonusOf(t, st, 1); b.Multiplier != 1 || b.Neighbours != 0 {
		t.Fatalf("the daisy still has a bonus from a wilted neighbour: %+v", b)
	}
	if st.Plots[1].Bonus != nil {
		t.Fatalf("a wilted plant must not show a bonus: %+v", st.Plots[1].Bonus)
	}
}

// Decision 11: a neighbour that is still growing gives nothing; once it matures the bonus starts.
func TestAGrowingNeighbourGivesNoBonusUntilItMatures(t *testing.T) {
	e := newFreshEnv(t)
	plotsFor(t, e, 2)
	dbExec(t, e, `UPDATE players SET silo_level = 4`)
	unlock(t, e, "tomato")
	mature(t, e, 1, "daisy", 0, 24)
	e.expect(201, "POST", "/api/pomodoros", service.PlantRequest{PlotID: 2, PlantType: "tomato"}) // 25 min
	e.clock.Advance(20 * time.Minute)
	if b := bonusOf(t, e.state(), 1); b.Multiplier != 1 {
		t.Fatalf("while the tomato grows: %+v", b)
	}
	e.clock.Advance(10 * time.Minute) // the tomato matured 5 min ago
	st := e.state()
	if b := bonusOf(t, st, 1); math.Abs(b.Multiplier-1.1) > 1e-9 {
		t.Fatalf("after the tomato matured: %+v", b)
	}
	// production before the tomato matured had no bonus; after it, a little
	d := baseRatePerHour("daisy", 24)
	wantMilli := d*(25.0/60)*1000 + d*(5.0/60)*1.1*1000 + baseRatePerHour("tomato", 36)*(5.0/60)*1.1*1000
	near(t, "Silo after 30 min", st.Silo.ContentMilli, int64(wantMilli), 3)
}

func TestClearingTheNeighbourEndsTheBonusAtThatMoment(t *testing.T) {
	e := newFreshEnv(t)
	plotsFor(t, e, 2)
	dbExec(t, e, `UPDATE players SET silo_level = 4`)
	mature(t, e, 1, "daisy", 0, 24)
	mature(t, e, 2, "tomato", 0, 36)
	e.clock.Advance(4 * time.Hour)
	e.expect(200, "POST", "/api/plots/2/clear", map[string]bool{"confirm": true})
	e.clock.Advance(6 * time.Hour)
	st := e.state()
	want := baseRatePerHour("daisy", 24)*(4*1.1+6) + baseRatePerHour("tomato", 36)*4*1.1
	near(t, "Silo", st.Silo.ContentMilli, int64(want*1000), 4)
}

func TestPrestigeMultipliesProduction(t *testing.T) {
	a, b := newFreshEnv(t), newFreshEnv(t)
	dbExec(t, b, `UPDATE players SET season = 3`) // +10% per completed season: ×1.2
	for _, e := range []*env{a, b} {
		mature(t, e, 1, "daisy", 0, 24)
		e.clock.Advance(5 * time.Hour)
	}
	x, _ := milli(t, a)
	y, _ := milli(t, b)
	near(t, "season 3 earns 20% more", y, int64(float64(x)*1.2), 3)
}

// The game settles on every request: counting in many steps must equal counting once, with neighbours maturing and wilting in between.
func TestSettlingOftenEqualsOnceWithSynergiesThroughTheAPI(t *testing.T) {
	build := func() *env {
		e := newFreshEnv(t)
		plotsFor(t, e, 4)
		dbExec(t, e, `UPDATE players SET silo_level = 4`)
		mature(t, e, 1, "daisy", 2*time.Hour, 24)
		mature(t, e, 2, "tomato", 0, 9) // wilts after 9 h
		mature(t, e, 3, "oak", -3*time.Hour, 40)
		dbExec(t, e, `UPDATE plots SET collected_to = matured_at WHERE id = 3`)
		mature(t, e, 4, "sunflower", 0, 20)
		return e
	}
	once, often := build(), build()
	once.clock.Advance(30 * time.Hour)
	for i := 0; i < 120; i++ {
		often.clock.Advance(15 * time.Minute)
		often.state()
	}
	a, _ := milli(t, once)
	b, _ := milli(t, often)
	near(t, "30 h in 120 requests vs 1", b, a, 3)
}

func TestSeedsListTheirCompatibleCrops(t *testing.T) {
	e := newFreshEnv(t)
	for _, s := range e.state().Seeds {
		if len(s.Compatible) != 2 {
			t.Fatalf("%s has %d compatible crops", s.Key, len(s.Compatible))
		}
		for _, other := range s.Compatible {
			if !game.Compatible(s.Key, other) {
				t.Errorf("%s lists %s, which is not its neighbour", s.Key, other)
			}
		}
	}
}
