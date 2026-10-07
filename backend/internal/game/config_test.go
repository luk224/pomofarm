package game

import (
	"encoding/json"
	"math"
	"os"
	"strconv"
	"testing"
)

// xlsx mirrors testdata/balance_xlsx.json, produced by tools/export_balance.py.
type xlsx struct {
	Params map[string]float64 `json:"params"`
	Crops  []struct {
		Name        string  `json:"name"`
		DurationMin int     `json:"duration_min"`
		Unlock      int     `json:"unlock"`
		Reward      int     `json:"reward"`
		LifeH       int     `json:"life_h"`
		E           float64 `json:"e"`
		YieldCoins  float64 `json:"yield_coins"`
		RatePerH    float64 `json:"rate_per_h"`
	} `json:"crops"`
	Flow []struct {
		DurationMin int     `json:"duration_min"`
		Reward      int     `json:"reward"`
		LifeH       int     `json:"life_h"`
		YieldCoins  float64 `json:"yield_coins"`
	} `json:"flow"`
	PlotCosts map[string]int `json:"plot_costs"`
	Silo      []struct {
		CapacityH int `json:"capacity_h"`
		Cost      int `json:"cost"`
	} `json:"silo"`
	Structures map[string]int `json:"structures"`
}

func loadXlsx(t *testing.T) xlsx {
	t.Helper()
	b, err := os.ReadFile("testdata/balance_xlsx.json")
	if err != nil {
		t.Fatal(err)
	}
	var x xlsx
	if err := json.Unmarshal(b, &x); err != nil {
		t.Fatal(err)
	}
	return x
}

func near(a, b, tol float64) bool { return math.Abs(a-b) <= tol }

func TestParamsMatchXlsx(t *testing.T) {
	x := loadXlsx(t)
	for name, got := range map[string]float64{
		"E0": E0, "BETA": Beta, "D0": D0, "FLOW_R_BASE": FlowRewardBase, "FLOW_EXP": FlowExp,
		"FLOW_L_BASE": FlowLifeBaseH, "FLOW_REF_D": FlowRefMin, "PLOT_BASE": PlotBase,
		"PLOT_GROWTH": PlotGrowth, "MAX_PLOTS": MaxPlots, "M_BEES": BeesBonus,
		"M_CAP": PlotMultiplierCap, "HIVE_BASE": HiveBase, "HIVE_GROWTH": HiveGrowth,
		"DOG_COST": DogCost, "DOG_SILO_H": DogSiloBonus, "UNLOCK_BEES": UnlockBees,
		"UNLOCK_DOG": UnlockDog, "PRESTIGE_BONUS": PrestigeBonus,
	} {
		if want, ok := x.Params[name]; !ok || !near(got, want, 1e-9) {
			t.Errorf("%s = %v, xlsx = %v (present=%v)", name, got, want, ok)
		}
	}
}

func TestCropsMatchXlsx(t *testing.T) {
	x := loadXlsx(t)
	if len(x.Crops) != len(Crops) {
		t.Fatalf("crops: %d in config, %d in xlsx", len(Crops), len(x.Crops))
	}
	for i, c := range Crops {
		w := x.Crops[i]
		if c.DurationMin != w.DurationMin || c.Unlock != w.Unlock || c.Reward != w.Reward || c.LifeH != w.LifeH {
			t.Errorf("%s: config %+v != xlsx %+v", c.Key, c, w)
		}
		if !near(YieldPerMin(float64(c.DurationMin)), w.E, 1e-9) {
			t.Errorf("%s e(d) = %v, xlsx %v", c.Key, YieldPerMin(float64(c.DurationMin)), w.E)
		}
		if !near(CycleYield(float64(c.DurationMin)), w.YieldCoins, 1e-9) {
			t.Errorf("%s Y = %v, xlsx %v", c.Key, CycleYield(float64(c.DurationMin)), w.YieldCoins)
		}
		if !near(RatePerHour(c.DurationMin, c.LifeH), w.RatePerH, 1e-9) {
			t.Errorf("%s g = %v, xlsx %v", c.Key, RatePerHour(c.DurationMin, c.LifeH), w.RatePerH)
		}
	}
}

func TestFlowMatchesXlsx(t *testing.T) {
	x := loadXlsx(t)
	if len(x.Flow) != 5 {
		t.Fatalf("flow rows in xlsx = %d, want 5", len(x.Flow))
	}
	for _, w := range x.Flow {
		if got := FlowReward(w.DurationMin); got != w.Reward {
			t.Errorf("flow %d min: reward %d, xlsx %d", w.DurationMin, got, w.Reward)
		}
		if got := FlowLifeH(w.DurationMin); got != w.LifeH {
			t.Errorf("flow %d min: life %d h, xlsx %d", w.DurationMin, got, w.LifeH)
		}
		if !near(CycleYield(float64(w.DurationMin)), w.YieldCoins, 1e-9) {
			t.Errorf("flow %d min: yield %v, xlsx %v", w.DurationMin, CycleYield(float64(w.DurationMin)), w.YieldCoins)
		}
	}
}

func TestPlotsSiloStructuresMatchXlsx(t *testing.T) {
	x := loadXlsx(t)
	for n := 2; n <= MaxPlots; n++ {
		if want := x.PlotCosts[strconv.Itoa(n)]; PlotCost(n) != want {
			t.Errorf("plot %d: %d, xlsx %d", n, PlotCost(n), want)
		}
	}
	if len(x.Silo) != len(Silo) {
		t.Fatalf("silo levels: %d vs xlsx %d", len(Silo), len(x.Silo))
	}
	for i, s := range Silo {
		if s.CapacityH != x.Silo[i].CapacityH || s.Cost != x.Silo[i].Cost {
			t.Errorf("silo %d: %+v vs xlsx %+v", i, s, x.Silo[i])
		}
	}
	for i := 0; i < MaxHives; i++ {
		if want := x.Structures["Colmena "+strconv.Itoa(i+1)]; HiveCost(i) != want {
			t.Errorf("hive %d: %d, xlsx %d", i+1, HiveCost(i), want)
		}
	}
	if x.Structures["Perro Pastor"] != DogCost {
		t.Errorf("dog cost %d, xlsx %d", DogCost, x.Structures["Perro Pastor"])
	}
	for name, got := range map[string]int{
		"Camino de piedra (casilla)": PathCost, "Valla (tramo)": FenceCost,
		"Farolillo": LanternCost, "Sombrero de paja del perro": DogHatCost,
	} {
		if x.Structures[name] != got {
			t.Errorf("%s: %d, xlsx %d", name, got, x.Structures[name])
		}
	}
}

// Totals and examples quoted in the GDD (v2).
func TestGDDTotals(t *testing.T) {
	plots := 0
	for n := 2; n <= MaxPlots; n++ {
		plots += PlotCost(n)
	}
	if plots != 1167 {
		t.Errorf("15 plots = %d 💧, GDD 1167", plots)
	}
	silo := 0
	for _, s := range Silo {
		silo += s.Cost
	}
	if silo != 605 {
		t.Errorf("silo = %d 💧, GDD 605", silo)
	}
	auto := DogCost
	for i := 0; i < MaxHives; i++ {
		auto += HiveCost(i)
	}
	if auto != 62500 {
		t.Errorf("automation = %d coins, GDD 62500", auto)
	}
	// GDD §4.3 table: cycle yield in coins and Flow table.
	for d, want := range map[int]int{10: 20, 25: 104, 35: 191, 45: 300, 60: 503, 75: 752, 90: 1044, 105: 1378, 120: 1752} {
		if got := int(math.Round(CycleYield(float64(d)))); got != want {
			t.Errorf("Y(%d) = %d, GDD %d", d, got, want)
		}
	}
	for d, want := range map[int]int{60: 25, 75: 35, 90: 46, 105: 58, 120: 71} {
		if got := FlowReward(d); got != want {
			t.Errorf("flow reward(%d) = %d, GDD %d", d, got, want)
		}
	}
	// Roble yields 4.2× the per-minute return of the Margarita (GDD §4.3).
	if r := YieldPerMin(60) / YieldPerMin(10); !near(r, 4.19, 0.01) {
		t.Errorf("oak/daisy e ratio = %v, GDD ≈ 4.2", r)
	}
}

func TestRestMin(t *testing.T) {
	for d, want := range map[int]int{10: 5, 25: 5, 26: 10, 45: 10, 46: 15, 120: 15} {
		if got := RestMin(d); got != want {
			t.Errorf("RestMin(%d) = %d, want %d", d, got, want)
		}
	}
}

func TestCoinsInMilli(t *testing.T) {
	if got := CycleYieldMilli(10); got != 20000 {
		t.Errorf("daisy cycle = %d milli, want 20000", got)
	}
	if _, ok := CropByKey("oak"); !ok {
		t.Error("CropByKey(oak) not found")
	}
}

func TestPlotOrderFillsTheGridOnceInTheDocumentedOrder(t *testing.T) {
	seen := map[[2]int]int{}
	for n := 1; n <= MaxPlots; n++ {
		x, y, ok := PlotPosition(n)
		if !ok || x < 0 || x > 3 || y < 0 || y > 3 {
			t.Fatalf("plot %d at (%d,%d) ok=%v is outside the 4×4 grid", n, x, y, ok)
		}
		if prev, dup := seen[[2]int{x, y}]; dup {
			t.Fatalf("plots %d and %d share cell (%d,%d)", prev, n, x, y)
		}
		seen[[2]int{x, y}] = n
	}
	if len(seen) != 16 {
		t.Fatalf("%d cells used, want 16", len(seen))
	}
	// the first plot is the one every new player starts with, and the first four make the central 2×2
	if x, y, _ := PlotPosition(1); x != 1 || y != 1 {
		t.Fatalf("first plot at (%d,%d), want (1,1)", x, y)
	}
	for n := 1; n <= 4; n++ {
		if x, y, _ := PlotPosition(n); x < 1 || x > 2 || y < 1 || y > 2 {
			t.Fatalf("plot %d at (%d,%d) is not in the central 2×2", n, x, y)
		}
	}
	if _, _, ok := PlotPosition(0); ok {
		t.Fatal("plot 0 exists")
	}
	if _, _, ok := PlotPosition(17); ok {
		t.Fatal("plot 17 exists")
	}
}
