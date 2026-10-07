package api

import (
	"fmt"
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/luk224/pomofarm/backend/internal/game"
	"github.com/luk224/pomofarm/backend/internal/service"
)

// invariantChecker verifies, after every random move, the rules that must hold whatever the player does.
type invariantChecker struct {
	t        *testing.T
	e        *env
	seed     int64
	step     int
	last     string
	prevCoin int64
	prevSilo int64
	prevCap  int64
	prevLife int64
	maxNow   time.Time
	// 🪙 can now be spent (hives, dog): a drop is allowed only by the exact price of a purchase that just succeeded.
	allowedDrop int64 // milli
	pendingCost int64 // milli: set before a purchase request; counts only if it succeeds
	spent       int64 // milli, all purchases so far
	grants      int64 // milli, 🪙 the test itself handed out
}

func (c *invariantChecker) fail(format string, args ...any) {
	c.t.Helper()
	c.t.Fatalf("seed %d, step %d (after %s): %s", c.seed, c.step, c.last, fmt.Sprintf(format, args...))
}

func (c *invariantChecker) check() {
	c.t.Helper()
	st := c.e.state()
	now := c.e.clock.T
	if now.After(c.maxNow) {
		c.maxNow = now
	}

	// money and points never go negative; coins only grow (nothing costs 🪙 yet); lifetime focus only grows and covers the balance
	if st.Player.FocusPoints < 0 || st.Player.CoinsMilli < 0 || st.Silo.ContentMilli < 0 {
		c.fail("negative balance: %+v silo %d", st.Player, st.Silo.ContentMilli)
	}
	if st.Player.CoinsMilli < c.prevCoin-c.allowedDrop {
		c.fail("coins went down by more than a purchase: %d -> %d (allowed drop %d)", c.prevCoin, st.Player.CoinsMilli, c.allowedDrop)
	}
	c.allowedDrop = 0
	if st.Player.LifetimeFocus < c.prevLife || st.Player.LifetimeFocus < st.Player.FocusPoints {
		c.fail("lifetime focus %d (before %d, balance %d)", st.Player.LifetimeFocus, c.prevLife, st.Player.FocusPoints)
	}
	// the Silo never exceeds its capacity (unless it already held more, e.g. capacity can only shrink if the player removed plants)
	if st.Silo.ContentMilli > c.prevSilo && st.Silo.ContentMilli > st.Silo.CapacityMilli+2 {
		c.fail("Silo %d above capacity %d (was %d)", st.Silo.ContentMilli, st.Silo.CapacityMilli, c.prevSilo)
	}
	c.prevCoin, c.prevSilo, c.prevCap, c.prevLife = st.Player.CoinsMilli, st.Silo.ContentMilli, st.Silo.CapacityMilli, st.Player.LifetimeFocus

	// plots: at most 16, unique cells, consistent states
	if len(st.Plots) > game.MaxPlots || st.Shop.PlotsOwned != len(st.Plots) {
		c.fail("%d plots, shop says %d", len(st.Plots), st.Shop.PlotsOwned)
	}
	cells := map[[2]int]bool{}
	growing := 0
	for _, p := range st.Plots {
		if cells[[2]int{p.X, p.Y}] {
			c.fail("two plots at (%d,%d)", p.X, p.Y)
		}
		cells[[2]int{p.X, p.Y}] = true
		switch p.State {
		case "empty":
			if p.PlantType != nil || p.Harvested || p.Bonus != nil {
				c.fail("empty plot %d with data %+v", p.ID, p)
			}
		case "growing":
			growing++
			if p.Bonus != nil || p.Harvested {
				c.fail("growing plot %d: %+v", p.ID, p)
			}
		case "mature":
			if p.WiltsAt == nil || p.MaturedAt == nil {
				c.fail("mature plot %d without dates", p.ID)
			}
			if w, _ := time.Parse(time.RFC3339Nano, *p.WiltsAt); !w.After(now) {
				c.fail("plot %d is mature but its life ended at %s (now %s)", p.ID, w, now)
			}
			// A plant stored as mature may carry a maturity time later than "now" if the system clock jumped back:
			// by that clock it has not matured yet, so it produces nothing (and shows no bonus) until the clock catches up.
			if m, _ := time.Parse(time.RFC3339Nano, *p.MaturedAt); !m.After(now) && p.Bonus == nil {
				c.fail("mature plot %d (matured %s, now %s) has no bonus block", p.ID, m, now)
			}
			if p.Bonus != nil && (p.Bonus.Multiplier < 1 || p.Bonus.Multiplier > 2*(1+game.PrestigeBonus*float64(st.Player.Season-1))*1.0001) {
				c.fail("mature plot %d bonus %+v", p.ID, p.Bonus)
			}
		case "withered":
			// Stored as withered, but a clock that jumped back before wilts_at makes it alive again for display purposes.
			if w, _ := time.Parse(time.RFC3339Nano, *p.WiltsAt); p.Bonus != nil && !w.After(now) {
				c.fail("withered plot %d still has a bonus", p.ID)
			}
		default:
			c.fail("plot %d in unknown state %q", p.ID, p.State)
		}
		if p.Harvested && p.State != "mature" && p.State != "withered" {
			c.fail("plot %d harvested but %s", p.ID, p.State)
		}
	}

	// the single active Pomodoro owns exactly the one growing plot
	if st.Pomodoro != nil {
		if growing != 1 || st.Pomodoro.PlotID == nil {
			c.fail("active Pomodoro but %d growing plots", growing)
		}
		if st.Pomodoro.RemainingMs < 0 || st.Pomodoro.RemainingMs > st.Pomodoro.PlannedS*1000 {
			c.fail("remaining %d ms of %d s", st.Pomodoro.RemainingMs, st.Pomodoro.PlannedS)
		}
	} else if growing != 0 {
		c.fail("%d plots growing with no active Pomodoro", growing)
	}
	var active int
	c.e.db.QueryRow(`SELECT COUNT(*) FROM pomodoros WHERE status IN ('running','paused')`).Scan(&active)
	if active > 1 {
		c.fail("%d active Pomodoros", active)
	}

	// hives: at most four, one per owned plot cell, and the bee flag matches their 3×3 coverage; a single dog
	if len(st.Automation.Bees.Hives) > game.MaxHives {
		c.fail("%d hives", len(st.Automation.Bees.Hives))
	}
	var hiveCells [][2]int
	for _, h := range st.Automation.Bees.Hives {
		if !cells[[2]int{h.X, h.Y}] || h.PlotID == 0 {
			c.fail("hive %d stands on (%d,%d), which is not one of the player's plots", h.ID, h.X, h.Y)
		}
		hiveCells = append(hiveCells, [2]int{h.X, h.Y})
	}
	cover := game.BeeCoverage(hiveCells)
	for _, p := range st.Plots {
		if p.Bonus != nil && p.Bonus.Bees != cover[[2]int{p.X, p.Y}] {
			c.fail("plot (%d,%d) bees=%v but its coverage is %v", p.X, p.Y, p.Bonus.Bees, cover[[2]int{p.X, p.Y}])
		}
	}
	var dogs int
	c.e.db.QueryRow(`SELECT COUNT(*) FROM structures WHERE kind = 'dog'`).Scan(&dogs)
	if dogs > 1 || (dogs == 1) != st.Automation.Dog.Owned {
		c.fail("%d dogs, state says owned=%v", dogs, st.Automation.Dog.Owned)
	}

	// rest never longer than the longest configurable one
	if st.Rest != nil && (st.Rest.RemainingMs < 0 || st.Rest.RemainingMs > 60*60*1000) {
		c.fail("rest %+v", st.Rest)
	}

	// each plant's counted-up-to marker stays between its maturity and its death
	rows, err := c.e.db.Query(`SELECT id, matured_at, wilts_at, collected_to FROM plots WHERE matured_at IS NOT NULL`)
	if err != nil {
		c.fail("query: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var m, w, col string
		rows.Scan(&id, &m, &w, &col)
		mt, _ := time.Parse(time.RFC3339Nano, m)
		wt, _ := time.Parse(time.RFC3339Nano, w)
		ct, _ := time.Parse(time.RFC3339Nano, col)
		if ct.Before(mt.Add(-time.Second)) || ct.After(wt.Add(time.Second)) {
			c.fail("plot %d collected_to %s outside [%s, %s]", id, ct, mt, wt)
		}
	}

	// conservation: everything collected plus what waits in the Silo cannot exceed what the plants could ever make
	// (every completed Pomodoro yields at most one plant life, at the best possible multiplier ×2.0)
	var potential float64
	prs, _ := c.e.db.Query(`SELECT planned_s FROM pomodoros WHERE status = 'completed'`)
	for prs.Next() {
		var s int64
		prs.Scan(&s)
		potential += game.CycleYield(float64(s)/60) * game.PlotMultiplierCap
	}
	prs.Close()
	// what the player holds, plus what they spent, cannot exceed what the plants could make plus what the test gave
	held := float64(st.Player.CoinsMilli+st.Silo.ContentMilli+c.spent-c.grants) / 1000
	if held > potential+0.01 {
		c.fail("holds %.2f 🪙 but plants could only ever make %.2f", held, potential)
	}
}

// TestRandomPlayNeverBreaksTheInvariants plays thousands of random moves — valid and invalid, with the clock moving
// forward by seconds to days and sometimes backwards — and checks the rules of the game after every one.
func TestRandomPlayNeverBreaksTheInvariants(t *testing.T) {
	seeds := 20
	if testing.Short() {
		seeds = 3
	}
	crops := []string{"daisy", "tomato", "sunflower", "apple", "oak"}
	totalMoves := 0
	for seed := int64(1); seed <= int64(seeds); seed++ {
		rnd := rand.New(rand.NewSource(seed))
		e := newFreshEnv(t)
		unlock(t, e, crops...)
		c := &invariantChecker{t: t, e: e, seed: seed}
		move := func(label string, method, path string, body any) int {
			c.step++
			c.last = label
			code, _ := e.do(method, path, body)
			if code >= 500 {
				c.fail("server error %d on %s %s", code, method, path)
			}
			if code == 201 && c.pendingCost > 0 {
				c.allowedDrop, c.spent = c.pendingCost, c.spent+c.pendingCost
			}
			c.pendingCost = 0
			c.check()
			totalMoves++
			return code
		}
		plotIDs := func() []int64 {
			var ids []int64
			for _, p := range e.state().Plots {
				ids = append(ids, p.ID)
			}
			return ids
		}
		for i := 0; i < 160; i++ {
			ids := plotIDs()
			pick := ids[rnd.Intn(len(ids))]
			switch r := rnd.Intn(116); {
			case r < 22:
				req := service.PlantRequest{PlotID: pick, PlantType: crops[rnd.Intn(5)]}
				if req.PlantType == "oak" && rnd.Intn(2) == 0 {
					req.DurationMin = 60 + rnd.Intn(61)
				}
				if rnd.Intn(4) == 0 {
					req.Tag = []string{"a", "b", "tesis", ""}[rnd.Intn(4)]
				}
				move("plant", "POST", "/api/pomodoros", req)
			case r < 42:
				dur := []time.Duration{time.Second, 30 * time.Second, 5 * time.Minute, 20 * time.Minute, time.Hour, 6 * time.Hour, 30 * time.Hour, 80 * time.Hour}[rnd.Intn(8)]
				if rnd.Intn(40) == 0 {
					dur = -time.Duration(1+rnd.Intn(5)) * time.Hour // the system clock goes backwards
				}
				e.clock.Advance(dur)
				move(fmt.Sprintf("advance %v", dur), "GET", "/api/state", nil)
			case r < 52:
				move("harvest", "POST", fmt.Sprintf("/api/plots/%d/harvest", pick), nil)
			case r < 60:
				move("clear", "POST", fmt.Sprintf("/api/plots/%d/clear", pick), map[string]bool{"confirm": rnd.Intn(3) > 0})
			case r < 64:
				move("pause", "POST", "/api/pomodoros/active/pause", nil)
			case r < 68:
				move("resume", "POST", "/api/pomodoros/active/resume", nil)
			case r < 71:
				move("cancel", "POST", "/api/pomodoros/active/cancel", nil)
			case r < 78:
				move("collect", "POST", "/api/silo/collect", nil)
			case r < 83:
				dbExec(t, e, `UPDATE players SET focus_points = focus_points + 40, lifetime_focus = lifetime_focus + 40`) // earned elsewhere
				move("buy plot", "POST", "/api/plots", nil)
			case r < 87:
				dbExec(t, e, `UPDATE players SET focus_points = focus_points + 80, lifetime_focus = lifetime_focus + 80`)
				move("upgrade silo", "POST", "/api/silo/upgrade", nil)
			case r < 90:
				move("skip rest", "POST", "/api/rest/skip", nil)
			case r < 93:
				move("setting", "POST", "/api/settings", map[string]string{"key": []string{"rest_enabled", "rest_short_min"}[rnd.Intn(2)], "value": []string{"0", "1", "7", "99"}[rnd.Intn(4)]})
			case r < 96:
				move("unlock", "POST", "/api/unlocks", map[string]string{"kind": "seed", "key": crops[rnd.Intn(5)]})
			case r >= 100 && r < 103:
				dbExec(t, e, `UPDATE players SET coins_milli = coins_milli + 6000000`) // 6.000 🪙 earned elsewhere
				c.grants += 6_000_000
				move("grant coins", "GET", "/api/state", nil)
			case r >= 103 && r < 106:
				dbExec(t, e, `UPDATE players SET focus_points = focus_points + 130, lifetime_focus = lifetime_focus + 130`)
				move("unlock animal", "POST", "/api/unlocks", map[string]string{"kind": "animal", "key": []string{"bees", "dog"}[rnd.Intn(2)]})
			case r >= 106 && r < 110:
				if next := e.state().Automation.Bees.NextCost; next != nil {
					c.pendingCost = int64(*next) * 1000
				}
				move("buy hive", "POST", "/api/structures", map[string]any{"kind": "hive", "plot_id": pick})
			case r >= 110 && r < 112:
				c.pendingCost = int64(game.DogCost) * 1000
				move("buy dog", "POST", "/api/structures", map[string]any{"kind": "dog"})
			case r >= 112:
				if hs := e.state().Automation.Bees.Hives; len(hs) > 0 {
					move("move hive", "POST", fmt.Sprintf("/api/structures/%d/move", hs[rnd.Intn(len(hs))].ID), map[string]any{"plot_id": pick})
				} else {
					move("move hive (none)", "POST", "/api/structures/1/move", map[string]any{"plot_id": pick})
				}
			default:
				dbExec(t, e, `UPDATE players SET season = ?`, 1+rnd.Intn(3)) // prestige seasons
				move("season", "GET", "/api/state", nil)
			}
		}
	}
	t.Logf("%d random moves across %d games: every invariant held", totalMoves, seeds)
	_ = math.Abs
}
