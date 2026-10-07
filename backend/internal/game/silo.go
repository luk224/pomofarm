package game

import (
	"math"
	"sort"
	"time"
)

// MicroPerCoin: the Silo is counted in millionths of a coin so that many tiny settlements add up exactly.
const MicroPerCoin = 1_000_000

// Producer is one mature plant over the stretch of time that still has to be accounted for:
// [From, To] = [collected_to, min(now, wilts_at)].
type Producer struct {
	// MicroPerHour is its production rate while alive, already multiplied by synergies/prestige.
	MicroPerHour float64
	From, To     time.Time
}

// ProducerMicroPerHour is g = Y / life (GDD §4.3) scaled by the plot's multiplier, in micro-coins per hour.
func ProducerMicroPerHour(growS, lifeS int64, multiplier float64) float64 {
	if lifeS <= 0 {
		return 0
	}
	y := CycleYield(float64(growS) / 60)
	return y / (float64(lifeS) / 3600) * MicroPerCoin * multiplier
}

// SiloStock is what the player has stored in the Silo but not collected.
type SiloStock struct {
	Micro            int64 // contents, in millionths of a coin
	PeakMicroPerHour int64 // highest total rate since the Silo was last emptied
}

// Accrue adds the production of `producers` to the Silo, segment by segment, never exceeding capacity.
//
// The timeline is cut at every moment a plant starts or stops producing; inside each segment the rate
// is the sum of the plants alive in it (GDD §6.2). The capacity is capHours × the highest rate seen so far
// since the last collection. Two guarantees:
//   - coins already in the Silo are never taken away, even if capacity later shrinks;
//   - a "running" peak makes the result independent of how often the game settles: one 30-hour
//     settlement and a thousand short ones give the same Silo.
func (s SiloStock) Accrue(producers []Producer, capHours float64) SiloStock {
	var cuts []time.Time
	for _, p := range producers {
		if p.To.After(p.From) && p.MicroPerHour > 0 {
			cuts = append(cuts, p.From, p.To)
		}
	}
	if len(cuts) == 0 {
		return s
	}
	sort.Slice(cuts, func(i, j int) bool { return cuts[i].Before(cuts[j]) })

	content := float64(s.Micro)
	peak := s.PeakMicroPerHour
	for i := 0; i+1 < len(cuts); i++ {
		t0, t1 := cuts[i], cuts[i+1]
		if !t1.After(t0) {
			continue
		}
		var rate float64
		for _, p := range producers {
			if p.MicroPerHour > 0 && !p.From.After(t0) && !p.To.Before(t1) {
				rate += p.MicroPerHour
			}
		}
		if rate == 0 {
			continue
		}
		if r := int64(math.Round(rate)); r > peak {
			peak = r
		}
		capacity := capHours * float64(peak)
		if content < capacity { // a full Silo stops production; it never shrinks what is already stored
			content = math.Min(content+rate*t1.Sub(t0).Hours(), capacity)
		}
	}
	return SiloStock{Micro: int64(math.Round(content)), PeakMicroPerHour: peak}
}

// Capacity is the most the Silo can hold given the peak rate and the capacity in hours.
func (s SiloStock) Capacity(capHours float64) int64 {
	return int64(math.Round(capHours * float64(s.PeakMicroPerHour)))
}

// SiloCapacityHours is the Silo's capacity in hours of production for a level, with the Dog's bonus (GDD §4.5).
func SiloCapacityHours(level int, hasDog bool) float64 {
	if level < 0 {
		level = 0
	}
	if level >= len(Silo) {
		level = len(Silo) - 1
	}
	h := float64(Silo[level].CapacityH)
	if hasDog {
		h += DogSiloBonus
	}
	return h
}
