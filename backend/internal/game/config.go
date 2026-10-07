// Package game holds the pure game rules. config.go is the single source of
// balance numbers: it mirrors pomofarm_balance.xlsx (checked by config_test.go
// against tools/export_balance.py) and gdd_pomofarm_3d_v2.md §4.
package game

import "math"

// Coins are stored in thousandths (INTEGER) so no fractions are lost.
const MilliPerCoin = 1000

// Curve: e(d) = E0 · (d/D0)^Beta coins per focus minute; yield Y = e·d.
const (
	E0   = 2.0
	Beta = 0.8
	D0   = 10.0
)

// Crop is one seed type. Reward/Unlock are in 💧, LifeH is the useful life in hours.
type Crop struct {
	Key         string
	DurationMin int
	Unlock      int
	Reward      int
	LifeH       int
}

// Crops in ring order (GDD §4.6): each is compatible with its two neighbours,
// and Oak wraps around to Daisy.
var Crops = []Crop{
	{"daisy", 10, 0, 1, 24},
	{"tomato", 25, 8, 5, 36},
	{"sunflower", 35, 35, 8, 54},
	{"apple", 45, 100, 15, 72},
	{"oak", 60, 250, 25, 108},
}

// Flow mode (Oak, 60–120 min): reward = FlowRewardBase·(d/FlowRefMin)^FlowExp,
// life = FlowLifeBaseH·d/FlowRefMin.
const (
	FlowRewardBase = 25.0
	FlowExp        = 1.5
	FlowLifeBaseH  = 108
	FlowRefMin     = 60
	FlowMinMin     = 60
	FlowMaxMin     = 120
)

// Plots: plot n (2..MaxPlots) costs ceil(PlotBase·PlotGrowth^(n-2)) 💧; plot 1 is free.
const (
	PlotBase   = 3.0
	PlotGrowth = 1.4
	MaxPlots   = 16
)

// SiloLevel capacity is in hours of production; Cost in 💧.
type SiloLevel struct {
	CapacityH int
	Cost      int
}

var Silo = []SiloLevel{{12, 0}, {24, 25}, {36, 70}, {48, 160}, {72, 350}}

// Automation: unlocks cost 💧 (one-off), purchases cost coins.
const (
	UnlockBees   = 30
	UnlockDog    = 120
	HiveBase     = 4000 // coins for the 1st hive
	HiveGrowth   = 1.5
	MaxHives     = 4
	DogCost      = 30000 // coins
	DogSiloBonus = 12    // hours added to the Silo
)

// Synergies (GDD §4.6). Bonuses add up and are capped per plot.
const (
	AdjacencyPerNeighbour = 0.10
	AdjacencyCap          = 0.40
	FullGardenBonus       = 0.15
	BeesBonus             = 0.25
	PlotMultiplierCap     = 2.0
	PrestigeBonus         = 0.10 // per completed season, above the cap
)

// Decoration sink (coins). Not a prestige requirement.
const (
	PathCost    = 15
	FenceCost   = 25
	LanternCost = 80
	DogHatCost  = 600
)

// Rest after a Pomodoro (GDD §3.2), in minutes, by Pomodoro duration.
func RestMin(durationMin int) int {
	switch {
	case durationMin <= 25:
		return 5
	case durationMin <= 45:
		return 10
	default:
		return 15
	}
}

// YieldPerMin is e(d): coins per focus minute for a Pomodoro of d minutes.
func YieldPerMin(d float64) float64 { return E0 * math.Pow(d/D0, Beta) }

// CycleYield is Y = e(d)·d in coins.
func CycleYield(d float64) float64 { return YieldPerMin(d) * d }

// CycleYieldMilli is Y in thousandths of a coin, rounded.
func CycleYieldMilli(d int) int64 {
	return int64(math.Round(CycleYield(float64(d)) * MilliPerCoin))
}

// RatePerHour is g = Y / life, in coins per hour while the plant lives.
func RatePerHour(durationMin, lifeH int) float64 {
	return CycleYield(float64(durationMin)) / float64(lifeH)
}

// RateMilliPerSecond is the production rate of a plant in thousandths of a coin per second.
func RateMilliPerSecond(durationMin, lifeH int) float64 {
	return RatePerHour(durationMin, lifeH) * MilliPerCoin / 3600
}

// FlowReward is the 💧 reward of an Oak Flow session of d minutes (rounded).
func FlowReward(d int) int {
	return int(math.Round(FlowRewardBase * math.Pow(float64(d)/FlowRefMin, FlowExp)))
}

// FlowLifeH is the useful life in hours of an Oak Flow session of d minutes.
func FlowLifeH(d int) int { return FlowLifeBaseH * d / FlowRefMin }

// PlotCost is the 💧 cost of plot n (2..MaxPlots). Plot 1 is free (returns 0).
func PlotCost(n int) int {
	if n <= 1 {
		return 0
	}
	return int(math.Ceil(PlotBase * math.Pow(PlotGrowth, float64(n-2))))
}

// HiveCost is the coin cost of the i-th hive (0-based).
func HiveCost(i int) int { return int(math.Round(HiveBase * math.Pow(HiveGrowth, float64(i)))) }

// CropByKey returns the crop with the given key.
func CropByKey(key string) (Crop, bool) {
	for _, c := range Crops {
		if c.Key == key {
			return c, true
		}
	}
	return Crop{}, false
}
