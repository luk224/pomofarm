package game

import (
	"math"
	"math/rand"
	"testing"
	"time"
)

func cell(x, y int, crop string) Cell { return Cell{x, y, crop} }
func near2(a, b float64) bool         { return math.Abs(a-b) < 1e-9 }

func TestCompatibilityRing(t *testing.T) {
	// GDD §4.6: Daisy – Tomato – Sunflower – Apple – Oak – (back to Daisy)
	ring := []string{"daisy", "tomato", "sunflower", "apple", "oak"}
	for i, a := range ring {
		for j, b := range ring {
			d := (i - j + 5) % 5
			want := d == 1 || d == 4
			if got := Compatible(a, b); got != want {
				t.Errorf("Compatible(%s,%s) = %v, want %v", a, b, got, want)
			}
		}
	}
	if !Compatible("sunflower", "apple") {
		t.Error("the GDD's own example (Sunflower next to Apple) must be compatible")
	}
	if !Compatible("oak", "daisy") {
		t.Error("the ring must wrap around: Oak and Daisy are neighbours")
	}
	if Compatible("daisy", "ghost") || Compatible("ghost", "ghost") {
		t.Error("an unknown crop is compatible with nothing")
	}
	for crop, want := range map[string][2]string{"daisy": {"oak", "tomato"}, "tomato": {"daisy", "sunflower"}, "oak": {"apple", "daisy"}} {
		got := CompatibleWith(crop)
		if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
			t.Errorf("CompatibleWith(%s) = %v, want %v", crop, got, want)
		}
	}
}

func TestAdjacencyIsTenPercentPerCompatibleNeighbour(t *testing.T) {
	line := []Cell{cell(0, 0, "daisy"), cell(1, 0, "tomato"), cell(2, 0, "sunflower")}
	b := Synergies(line, nil)
	for pos, want := range map[[2]int]float64{{0, 0}: 1.1, {1, 0}: 1.2, {2, 0}: 1.1} {
		if !near2(b[pos].Total, want) {
			t.Errorf("%v: ×%.2f, want ×%.2f", pos, b[pos].Total, want)
		}
	}
	if b[[2]int{1, 0}].Neighbours != 2 || b[[2]int{1, 0}].Garden {
		t.Errorf("middle plant: %+v", b[[2]int{1, 0}])
	}
}

func TestOnlyOrthogonalNeighboursCount(t *testing.T) {
	diag := []Cell{cell(0, 0, "daisy"), cell(1, 1, "tomato")}
	for _, b := range Synergies(diag, nil) {
		if b.Total != 1 {
			t.Fatalf("a diagonal neighbour gave a bonus: %+v", b)
		}
	}
}

func TestNoMonocultureBonusAndNoPenalty(t *testing.T) {
	var four []Cell
	for _, p := range [][2]int{{0, 0}, {1, 0}, {0, 1}, {1, 1}} {
		four = append(four, cell(p[0], p[1], "daisy"))
	}
	for pos, b := range Synergies(four, nil) {
		if b.Total != 1 || b.Garden {
			t.Errorf("%v: same-crop block gives %+v; GDD: no bonus and no penalty", pos, b)
		}
	}
}

func TestAPlantAloneOrWithIncompatibleNeighboursGetsNothing(t *testing.T) {
	if b := Synergies([]Cell{cell(2, 2, "oak")}, nil)[[2]int{2, 2}]; b.Total != 1 {
		t.Errorf("alone: %+v", b)
	}
	two := Synergies([]Cell{cell(0, 0, "daisy"), cell(1, 0, "sunflower")}, nil) // daisy–sunflower are not ring neighbours
	if two[[2]int{0, 0}].Total != 1 || two[[2]int{1, 0}].Total != 1 {
		t.Errorf("incompatible pair: %+v", two)
	}
}

// Huerto completo: a 2×2 of four different crops gives +15% to each, on top of adjacency.
func TestFullGardenPattern(t *testing.T) {
	block := []Cell{cell(0, 0, "daisy"), cell(1, 0, "tomato"), cell(0, 1, "oak"), cell(1, 1, "sunflower")}
	b := Synergies(block, nil)
	for pos, want := range map[[2]int]float64{
		{0, 0}: 1 + 0.2 + 0.15, // daisy: tomato and oak next to it
		{1, 0}: 1 + 0.2 + 0.15, // tomato: daisy and sunflower
		{0, 1}: 1 + 0.1 + 0.15, // oak: daisy yes, sunflower no
		{1, 1}: 1 + 0.1 + 0.15, // sunflower: tomato yes, oak no
	} {
		if !near2(b[pos].Total, want) || !b[pos].Garden {
			t.Errorf("%v: %+v, want ×%.2f with garden", pos, b[pos], want)
		}
	}
}

func TestFullGardenNeedsFourDifferentCropsAllProducing(t *testing.T) {
	dup := Synergies([]Cell{cell(0, 0, "daisy"), cell(1, 0, "tomato"), cell(0, 1, "oak"), cell(1, 1, "tomato")}, nil)
	for pos, b := range dup {
		if b.Garden {
			t.Errorf("%v got the garden bonus with a repeated crop", pos)
		}
	}
	// the fourth plant is still growing / withered, so it is simply not in the list of producing cells
	three := Synergies([]Cell{cell(0, 0, "daisy"), cell(1, 0, "tomato"), cell(0, 1, "oak")}, nil)
	for pos, b := range three {
		if b.Garden {
			t.Errorf("%v got the garden bonus with only three producing plants", pos)
		}
	}
}

func TestOverlappingGardensPayOnlyOnce(t *testing.T) {
	// 2×3: two overlapping 2×2 blocks, each with four different crops
	farm := []Cell{
		cell(0, 0, "daisy"), cell(1, 0, "tomato"), cell(2, 0, "sunflower"),
		cell(0, 1, "apple"), cell(1, 1, "oak"), cell(2, 1, "daisy"),
	}
	b := Synergies(farm, nil)
	shared := b[[2]int{1, 0}] // tomato belongs to both blocks
	if !shared.Garden {
		t.Fatal("tomato should be in a garden")
	}
	if !near2(shared.Total, 1+shared.Adjacency+0.15) {
		t.Errorf("shared plant ×%.2f: the +15%% must be paid once, not per block (adjacency %.2f)", shared.Total, shared.Adjacency)
	}
}

func TestBestCaseAndTheCap(t *testing.T) {
	// daisy in the middle with four compatible neighbours and inside a garden of four different crops
	farm := []Cell{
		cell(1, 1, "daisy"), cell(2, 1, "tomato"), cell(1, 0, "oak"), cell(2, 0, "sunflower"),
		cell(0, 1, "tomato"), cell(1, 2, "oak"),
	}
	b := Synergies(farm, nil)[[2]int{1, 1}]
	if b.Neighbours != 4 || !b.Garden || !near2(b.Adjacency, 0.4) || !near2(b.Total, 1.55) {
		t.Fatalf("best case without bees: %+v, want 4 neighbours, +40%%, garden, ×1.55", b)
	}
	withBees := Synergies(farm, map[[2]int]bool{{1, 1}: true})[[2]int{1, 1}]
	if !near2(withBees.Total, 1.8) {
		t.Fatalf("with bees: ×%.2f, want ×1.80 (1 + 0.40 + 0.15 + 0.25)", withBees.Total)
	}
	for _, v := range Synergies(farm, map[[2]int]bool{{1, 1}: true, {2, 1}: true, {1, 0}: true}) {
		if v.Total > PlotMultiplierCap {
			t.Fatalf("multiplier ×%.2f exceeds the ×%.1f cap", v.Total, PlotMultiplierCap)
		}
	}
}

func TestBeesAddQuarterOnceAndOnlyToProducingPlants(t *testing.T) {
	b := Synergies([]Cell{cell(0, 0, "daisy")}, map[[2]int]bool{{0, 0}: true, {5, 5}: true})
	if !near2(b[[2]int{0, 0}].Total, 1.25) {
		t.Errorf("bees on a lone plant: %+v", b)
	}
	if _, ok := b[[2]int{5, 5}]; ok {
		t.Error("a covered cell with no producing plant got a bonus entry")
	}
}

// ---------- time-varying production ----------

var base = time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

func hrs(h float64) time.Time { return base.Add(time.Duration(h * float64(time.Hour))) }

func plot(id int64, x, y int, crop string, matured, wilts, collected float64) FarmPlot {
	c, _ := CropByKey(crop)
	return FarmPlot{ID: id, X: x, Y: y, Crop: crop, GrowS: int64(c.DurationMin) * 60, LifeS: int64(c.LifeH) * 3600,
		Matured: hrs(matured), Wilts: hrs(wilts), Collected: hrs(collected)}
}

func total(ps []Producer) float64 {
	var micro float64
	for _, p := range ps {
		micro += p.MicroPerHour * p.To.Sub(p.From).Hours()
	}
	return micro / MicroPerCoin
}

func baseRate(p FarmPlot) float64 { return ProducerMicroPerHour(p.GrowS, p.LifeS, 1) / MicroPerCoin }

func TestALonePlantProducesAsBefore(t *testing.T) {
	p := plot(1, 0, 0, "daisy", -3, 21, -3)
	got := total(FarmProducers([]FarmPlot{p}, hrs(5), 1, nil))
	if want := baseRate(p) * 8; math.Abs(got-want) > 1e-9 {
		t.Fatalf("got %.4f, want %.4f", got, want)
	}
}

func TestBonusAppearsWhenTheNeighbourMatures(t *testing.T) {
	a := plot(1, 0, 0, "daisy", 0, 24, 0)
	b := plot(2, 1, 0, "tomato", 5, 41, 5) // matures at 5 h
	got := total(FarmProducers([]FarmPlot{a, b}, hrs(10), 1, nil))
	want := baseRate(a)*(5+5*1.1) + baseRate(b)*5*1.1
	if math.Abs(got-want) > 1e-6 {
		t.Fatalf("got %.4f, want %.4f (daisy ×1.0 for 5 h then ×1.1; tomato ×1.1)", got, want)
	}
}

func TestBonusDisappearsWhenTheNeighbourWilts(t *testing.T) {
	a := plot(1, 0, 0, "daisy", 0, 100, 0)
	b := plot(2, 1, 0, "tomato", 0, 6, 0) // wilts at 6 h
	got := total(FarmProducers([]FarmPlot{a, b}, hrs(10), 1, nil))
	want := baseRate(a)*(6*1.1+4*1.0) + baseRate(b)*6*1.1
	if math.Abs(got-want) > 1e-6 {
		t.Fatalf("got %.4f, want %.4f", got, want)
	}
}

func TestAGrowingNeighbourGivesNothingUntilItMatures(t *testing.T) {
	a := plot(1, 0, 0, "daisy", 0, 24, 0)
	// the tomato would mature at 8 h; before that it is simply not part of the farm's producing plants
	b := plot(2, 1, 0, "tomato", 8, 44, 8)
	got := total(FarmProducers([]FarmPlot{a, b}, hrs(8), 1, nil)) // window ends exactly when it matures
	if want := baseRate(a) * 8; math.Abs(got-want) > 1e-9 {
		t.Fatalf("got %.4f, want %.4f: the bonus must not start before the neighbour produces", got, want)
	}
}

func TestPrestigeMultipliesAboveTheCap(t *testing.T) {
	a := plot(1, 0, 0, "daisy", 0, 24, 0)
	b := plot(2, 1, 0, "tomato", 0, 36, 0)
	plain := total(FarmProducers([]FarmPlot{a, b}, hrs(10), 1, nil))
	boosted := total(FarmProducers([]FarmPlot{a, b}, hrs(10), 1.2, nil))
	if math.Abs(boosted-plain*1.2) > 1e-6 {
		t.Fatalf("prestige ×1.2: %.4f vs %.4f", boosted, plain*1.2)
	}
}

func TestAlreadyCountedProductionIsNotCountedAgain(t *testing.T) {
	p := plot(1, 0, 0, "daisy", 0, 24, 10)
	if got := FarmProducers([]FarmPlot{p}, hrs(10), 1, nil); len(got) != 0 {
		t.Fatalf("nothing is pending, got %d pieces", len(got))
	}
	if got := FarmProducers(nil, hrs(10), 1, nil); got != nil {
		t.Fatal("empty farm")
	}
}

// The central guarantee: the game settles on every request, so counting in many small steps must equal
// counting once, even though the multipliers change whenever a neighbour matures or wilts.
func TestSettlingOftenEqualsOnceWithSynergies(t *testing.T) {
	rnd := rand.New(rand.NewSource(11))
	crops := []string{"daisy", "tomato", "sunflower", "apple", "oak"}
	for trial := 0; trial < 120; trial++ {
		var farm []FarmPlot
		used := map[[2]int]bool{}
		for i := 0; i < 2+rnd.Intn(9); i++ {
			x, y := rnd.Intn(4), rnd.Intn(4)
			if used[[2]int{x, y}] {
				continue
			}
			used[[2]int{x, y}] = true
			matured := rnd.Float64()*50 - 10
			life := 6 + rnd.Float64()*70
			c := crops[rnd.Intn(5)]
			p := plot(int64(i+1), x, y, c, matured, matured+life, 0)
			p.Collected = hrs(math.Max(matured, 0))
			farm = append(farm, p)
		}
		end := 90.0
		once := SiloStock{}.Accrue(FarmProducers(append([]FarmPlot(nil), farm...), hrs(end), 1.1, nil), 1e9)

		stepped := SiloStock{}
		steps := append([]FarmPlot(nil), farm...)
		for now := 0.0; now < end; {
			now = math.Min(end, now+0.05+rnd.Float64()*14)
			stepped = stepped.Accrue(FarmProducers(steps, hrs(now), 1.1, nil), 1e9)
			for i := range steps { // what the service does after counting: move each plant's marker forward
				to := steps[i].Wilts
				if hrs(now).Before(to) {
					to = hrs(now)
				}
				if to.After(steps[i].Collected) {
					steps[i].Collected = to
				}
			}
		}
		if d := math.Abs(float64(once.Micro-stepped.Micro)) / MicroPerCoin; d > 0.001 {
			t.Fatalf("trial %d: once %.5f 🪙, in steps %.5f 🪙 (diff %.5f) with farm %+v", trial, coins(once.Micro), coins(stepped.Micro), d, farm)
		}
	}
}

func TestCurrentBonusesOnlyForPlantsProducingNow(t *testing.T) {
	a := plot(1, 0, 0, "daisy", 0, 24, 0)
	b := plot(2, 1, 0, "tomato", 0, 6, 0)
	c := plot(3, 0, 1, "oak", 30, 100, 30) // not mature yet at 10 h
	got := CurrentBonuses([]FarmPlot{a, b, c}, hrs(10), nil)
	if len(got) != 1 || !near2(got[1].Total, 1) {
		t.Fatalf("at 10 h only the daisy produces (its tomato neighbour wilted at 6 h): %+v", got)
	}
	got = CurrentBonuses([]FarmPlot{a, b, c}, hrs(3), nil)
	if len(got) != 2 || !near2(got[1].Total, 1.1) || !near2(got[2].Total, 1.1) {
		t.Fatalf("at 3 h daisy and tomato help each other: %+v", got)
	}
}

func TestBeeCoverageIsAThreeByThreeAreaThatDoesNotStack(t *testing.T) {
	one := BeeCoverage([][2]int{{1, 1}})
	if len(one) != 9 || !one[[2]int{0, 0}] || !one[[2]int{2, 2}] || !one[[2]int{1, 1}] || one[[2]int{3, 1}] || one[[2]int{1, 3}] {
		t.Fatalf("one hive at (1,1) covers %v", one)
	}
	// GDD §4.7: four hives cover the 4×4 field; the central 2×2 is the layout that does it
	four := BeeCoverage([][2]int{{1, 1}, {2, 1}, {1, 2}, {2, 2}})
	for x := 0; x < 4; x++ {
		for y := 0; y < 4; y++ {
			if !four[[2]int{x, y}] {
				t.Errorf("cell (%d,%d) is not covered by four central hives", x, y)
			}
		}
	}
	// a hive in a corner covers only the cells that exist around it, plus off-grid cells that nothing uses
	if c := BeeCoverage([][2]int{{0, 0}}); !c[[2]int{0, 0}] || !c[[2]int{1, 1}] || c[[2]int{2, 0}] {
		t.Errorf("corner hive: %v", c)
	}
	if len(BeeCoverage(nil)) != 0 {
		t.Error("no hives, no coverage")
	}
}

func TestBeesFlagAndBonusInSynergies(t *testing.T) {
	b := Synergies([]Cell{cell(0, 0, "daisy")}, BeeCoverage([][2]int{{0, 0}}))[[2]int{0, 0}]
	if !b.Bees || !near2(b.Total, 1.25) {
		t.Fatalf("%+v", b)
	}
}
