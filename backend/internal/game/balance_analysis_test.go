package game

import (
	"math/rand"
	"testing"
)

// averageMultiplier is the mean per-plot multiplier of a full farm of n producing plants laid out as the game unlocks them.
func averageMultiplier(layout []int, bees bool) float64 {
	cells := make([]Cell, len(layout))
	beeMap := map[[2]int]bool{}
	for i, c := range layout {
		x, y, _ := PlotPosition(i + 1)
		cells[i] = Cell{x, y, Crops[c].Key}
		if bees {
			beeMap[[2]int{x, y}] = true
		}
	}
	var sum float64
	for _, b := range Synergies(cells, beeMap) {
		sum += b.Total
	}
	return sum / float64(len(layout))
}

// bestLayout searches crop assignments that maximise the average multiplier (simulated annealing, fixed seed).
func bestLayout(n int, bees bool) ([]int, float64) {
	rnd := rand.New(rand.NewSource(42))
	var best []int
	bestScore := 0.0
	for restart := 0; restart < 12; restart++ {
		cur := make([]int, n)
		for i := range cur {
			cur[i] = rnd.Intn(5)
		}
		score := averageMultiplier(cur, bees)
		temp := 0.05
		for it := 0; it < 6000; it++ {
			i := rnd.Intn(n)
			old := cur[i]
			cur[i] = rnd.Intn(5)
			s := averageMultiplier(cur, bees)
			if s >= score || rnd.Float64() < expApprox((s-score)/temp) {
				score = s
			} else {
				cur[i] = old
			}
			temp *= 0.999
			if score > bestScore {
				bestScore, best = score, append([]int(nil), cur...)
			}
		}
	}
	return best, bestScore
}

func expApprox(x float64) float64 { // e^x for x ≤ 0, enough for annealing
	if x < -20 {
		return 0
	}
	r := 1.0
	for i := 0; i < 12; i++ {
		r = 1 + x*r/float64(12-i)
	}
	if r < 0 {
		return 0
	}
	return r
}

// BALANCE ANALYSIS. sim.py assumes an average multiplier M = 1 + 0.15 (adjacency) + 0.10 (Huerto completo, from 8 plots)
// [+ 0.25 bees] = 1.25, or 1.50 with bees. This asks what the real rules allow, and what a "careless" farm earns.
func TestAchievableAverageMultiplier(t *testing.T) {
	t.Logf("%-8s %-22s %-22s %-22s", "plots", "best layout (no bees)", "best with all bees", "random layout (avg of 200)")
	rnd := rand.New(rand.NewSource(7))
	for _, n := range []int{4, 8, 12, 16} {
		_, no := bestLayout(n, false)
		_, with := bestLayout(n, true)
		var rsum float64
		for i := 0; i < 200; i++ {
			l := make([]int, n)
			for j := range l {
				l[j] = rnd.Intn(5)
			}
			rsum += averageMultiplier(l, false)
		}
		t.Logf("%-8d ×%.3f                ×%.3f                ×%.3f", n, no, with, rsum/200)
		if no < 1 || with > PlotMultiplierCap || no > 1+AdjacencyCap+FullGardenBonus+1e-9 {
			t.Errorf("n=%d: impossible values no=%.3f with=%.3f", n, no, with)
		}
		if with < no {
			t.Errorf("n=%d: bees made it worse (%.3f < %.3f)", n, with, no)
		}
	}
}
