package game

import (
	"sort"
	"time"
)

// Cell is a PRODUCING plant (mature, not withered) at a grid position. Only producing plants give or receive
// synergies (GDD §4.6, decision 11): a neighbour that is still growing or already withered does nothing.
type Cell struct {
	X, Y int
	Crop string
}

// Bonus is what one producing plot earns from its surroundings.
type Bonus struct {
	Neighbours int     // orthogonal neighbours that are compatible, up to 4
	Adjacency  float64 // +10% each, capped at +40%
	Garden     bool    // part of a 2×2 block of four different crops
	Total      float64 // 1 + adjacency + garden (+ bees), capped at PlotMultiplierCap; prestige is applied above it
}

// ringIndex is each crop's place in the compatibility ring Daisy–Tomato–Sunflower–Apple–Oak–(Daisy).
func ringIndex(crop string) int {
	for i, c := range Crops {
		if c.Key == crop {
			return i
		}
	}
	return -1
}

// Compatible reports whether two crops are neighbours in the ring. A crop is not compatible with itself:
// planting the same crop together earns nothing, and costs nothing either (no monoculture penalty).
func Compatible(a, b string) bool {
	ia, ib := ringIndex(a), ringIndex(b)
	if ia < 0 || ib < 0 {
		return false
	}
	n := len(Crops)
	d := (ia - ib + n) % n
	return d == 1 || d == n-1
}

// CompatibleWith lists the two crops a crop combines with, for showing on the seed packets.
func CompatibleWith(crop string) []string {
	i := ringIndex(crop)
	if i < 0 {
		return nil
	}
	n := len(Crops)
	return []string{Crops[(i+n-1)%n].Key, Crops[(i+1)%n].Key}
}

// Synergies computes the bonus of every producing cell (GDD §4.6):
//   - adjacency: +10% for each orthogonal neighbour whose crop is compatible, at most +40%;
//   - Huerto completo: +15% if the cell belongs to any 2×2 block holding four different crops (once, not per block);
//   - bees: +25% to cells in `bees` (3×3 areas; they do not stack), supplied by the caller;
//   - the three add up, then the total is capped at ×2.0 per plot.
func Synergies(cells []Cell, bees map[[2]int]bool) map[[2]int]Bonus {
	at := make(map[[2]int]string, len(cells))
	for _, c := range cells {
		at[[2]int{c.X, c.Y}] = c.Crop
	}
	garden := map[[2]int]bool{}
	for _, c := range cells {
		// the block whose top-left corner is c
		block := [4][2]int{{c.X, c.Y}, {c.X + 1, c.Y}, {c.X, c.Y + 1}, {c.X + 1, c.Y + 1}}
		seen := map[string]bool{}
		for _, p := range block {
			if crop, ok := at[p]; ok {
				seen[crop] = true
			}
		}
		if len(seen) == 4 {
			for _, p := range block {
				garden[p] = true
			}
		}
	}
	out := make(map[[2]int]Bonus, len(cells))
	for _, c := range cells {
		pos := [2]int{c.X, c.Y}
		n := 0
		for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			if other, ok := at[[2]int{c.X + d[0], c.Y + d[1]}]; ok && Compatible(c.Crop, other) {
				n++
			}
		}
		b := Bonus{Neighbours: n, Garden: garden[pos]}
		b.Adjacency = minFloat(AdjacencyPerNeighbour*float64(n), AdjacencyCap)
		total := 1 + b.Adjacency
		if b.Garden {
			total += FullGardenBonus
		}
		if bees[pos] {
			total += BeesBonus
		}
		b.Total = minFloat(total, PlotMultiplierCap)
		out[pos] = b
	}
	return out
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// FarmPlot is one plot with a mature plant, as stored.
type FarmPlot struct {
	ID        int64
	X, Y      int
	Crop      string
	GrowS     int64
	LifeS     int64
	Matured   time.Time
	Wilts     time.Time
	Collected time.Time // production is already counted up to here
}

func (p FarmPlot) aliveDuring(t0, t1 time.Time) bool {
	return !p.Matured.After(t0) && !p.Wilts.Before(t1)
}

// FarmProducers turns the farm into the pieces of production still to be counted, up to `now`.
//
// A plant's multiplier depends on which neighbours are producing, and that changes whenever any plant matures
// or wilts. So the timeline is cut at every such moment (not just at the plant's own), and each piece gets the
// multiplier valid inside it. Because a piece's multiplier depends only on the state of the farm at that
// moment, settling once or in many steps gives the same result.
// `prestige` multiplies on top of the per-plot cap (GDD §4.6); `bees` marks plots covered by beehives.
func FarmProducers(plots []FarmPlot, now time.Time, prestige float64, bees map[[2]int]bool) []Producer {
	var start time.Time
	have := false
	for _, p := range plots {
		to := p.Wilts
		if now.Before(to) {
			to = now
		}
		if !to.After(p.Collected) {
			continue // nothing left to count for this plant
		}
		if !have || p.Collected.Before(start) {
			start, have = p.Collected, true
		}
	}
	if !have {
		return nil
	}
	cuts := []time.Time{start, now}
	for _, p := range plots {
		for _, t := range []time.Time{p.Matured, p.Wilts, p.Collected} {
			if t.After(start) && t.Before(now) {
				cuts = append(cuts, t)
			}
		}
	}
	sort.Slice(cuts, func(i, j int) bool { return cuts[i].Before(cuts[j]) })

	var out []Producer
	for i := 0; i+1 < len(cuts); i++ {
		t0, t1 := cuts[i], cuts[i+1]
		if !t1.After(t0) {
			continue
		}
		var alive []FarmPlot
		var cells []Cell
		for _, p := range plots {
			if p.aliveDuring(t0, t1) {
				alive = append(alive, p)
				cells = append(cells, Cell{p.X, p.Y, p.Crop})
			}
		}
		bonus := Synergies(cells, bees)
		for _, p := range alive {
			if p.Collected.After(t0) {
				continue // this stretch of the plant was already counted
			}
			m := bonus[[2]int{p.X, p.Y}].Total * prestige
			out = append(out, Producer{MicroPerHour: ProducerMicroPerHour(p.GrowS, p.LifeS, m), From: t0, To: t1})
		}
	}
	return out
}

// CurrentBonuses is the bonus of every plot producing at `now`, for display and for the current rate.
func CurrentBonuses(plots []FarmPlot, now time.Time, bees map[[2]int]bool) map[int64]Bonus {
	var cells []Cell
	var alive []FarmPlot
	for _, p := range plots {
		if !p.Matured.After(now) && p.Wilts.After(now) {
			alive = append(alive, p)
			cells = append(cells, Cell{p.X, p.Y, p.Crop})
		}
	}
	b := Synergies(cells, bees)
	out := make(map[int64]Bonus, len(alive))
	for _, p := range alive {
		out[p.ID] = b[[2]int{p.X, p.Y}]
	}
	return out
}
