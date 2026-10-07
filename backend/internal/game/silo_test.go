package game

import (
	"math"
	"math/rand"
	"testing"
	"time"
)

var epoch = time.Date(2026, 3, 1, 8, 0, 0, 0, time.UTC)

func at(h float64) time.Time { return epoch.Add(time.Duration(h * float64(time.Hour))) }

func coins(micro int64) float64 { return float64(micro) / MicroPerCoin }

// GDD §6.2 worked example: M = 1, 30 h away, Tomatoes (wilt at 6 h), Sunflower (20 h), Apple (alive all 30 h).
func example62() []Producer {
	rate := func(minutes, lifeH int) float64 { return ProducerMicroPerHour(int64(minutes)*60, int64(lifeH)*3600, 1) }
	return []Producer{
		{MicroPerHour: rate(25, 36), From: at(0), To: at(6)},
		{MicroPerHour: rate(35, 54), From: at(0), To: at(20)},
		{MicroPerHour: rate(45, 72), From: at(0), To: at(30)},
	}
}

func TestGDDExampleRates(t *testing.T) {
	// per-hour rates quoted in the GDD: 2.89, 3.53 and 4.16 🪙/h
	ps := example62()
	for i, want := range []float64{2.89, 3.53, 4.16} {
		if got := ps[i].MicroPerHour / MicroPerCoin; math.Abs(got-want) > 0.01 {
			t.Errorf("plant %d rate = %.3f, GDD %.2f", i, got, want)
		}
	}
}

// GDD §6.2 + §8: "Ejemplo de 6.2 reproducido en una prueba automática, con los dos topes de Silo."
func TestGDDExampleSiloCaps(t *testing.T) {
	// The GDD worked the example with rates rounded to two decimals (10.58, 7.69, 4.16 🪙/h -> 212.7);
	// with the exact rates of §4.3 the same example gives 212.89. Both must hold.
	rounded := []Producer{
		{MicroPerHour: 2.89 * MicroPerCoin, From: at(0), To: at(6)},
		{MicroPerHour: 3.53 * MicroPerCoin, From: at(0), To: at(20)},
		{MicroPerHour: 4.16 * MicroPerCoin, From: at(0), To: at(30)},
	}
	if got := coins(SiloStock{}.Accrue(rounded, 24).Micro); math.Abs(got-212.7) > 0.06 {
		t.Errorf("GDD rounded rates, Silo 24 h: %.2f 🪙, GDD 212.7", got)
	}
	if got := coins(SiloStock{}.Accrue(rounded, 12).Micro); math.Abs(got-127.0) > 0.06 {
		t.Errorf("GDD rounded rates, Silo 12 h: %.2f 🪙, GDD 127.0", got)
	}
	cases := []struct {
		name     string
		capHours float64
		want     float64
		tol      float64
	}{
		{"exacto, sin tope (Silo de sobra)", 1000, 212.89, 0.01},
		{"exacto, Silo de 24 h: no trunca", 24, 212.89, 0.01},
		{"exacto, Silo de 12 h: se llena a las ≈14,3 h", 12, 127.0, 0.1},
	}
	for _, c := range cases {
		got := coins(SiloStock{}.Accrue(example62(), c.capHours).Micro)
		if math.Abs(got-c.want) > c.tol {
			t.Errorf("%s: %.3f 🪙, esperado %.2f", c.name, got, c.want)
		}
	}
	if got := coins(SiloStock{}.Accrue(example62(), 24).Micro); math.Abs(got-212.7) > 0.2 {
		t.Errorf("exact rates are %.2f, more than 0.2 from the GDD's 212.7", got)
	}
	// Cap quoted in the GDD: 24 × 10.58 = 253.9 and 12 × 10.58 = 127.0
	s := SiloStock{}.Accrue(example62(), 12)
	if got := coins(s.Capacity(12)); math.Abs(got-127.0) > 0.1 {
		t.Errorf("capacity 12 h = %.2f, GDD 127.0", got)
	}
	if got := coins(SiloStock{}.Accrue(example62(), 24).Capacity(24)); math.Abs(got-253.9) > 0.2 {
		t.Errorf("capacity 24 h = %.2f, GDD 253.9", got)
	}
}

func TestSiloFullStopsProductionButKeepsContents(t *testing.T) {
	full := SiloStock{Micro: 100 * MicroPerCoin, PeakMicroPerHour: 10 * MicroPerCoin} // cap at 10 h = 100 🪙: already full
	p := []Producer{{MicroPerHour: 10 * MicroPerCoin, From: at(0), To: at(50)}}
	got := full.Accrue(p, 10)
	if got.Micro != full.Micro {
		t.Fatalf("a full Silo changed: %.2f -> %.2f", coins(full.Micro), coins(got.Micro))
	}
}

func TestSiloNeverTakesAwayCoins(t *testing.T) {
	// capacity would be 2 h × 1 🪙/h = 2 🪙, but 50 are already stored: they stay
	s := SiloStock{Micro: 50 * MicroPerCoin, PeakMicroPerHour: 1 * MicroPerCoin}
	got := s.Accrue([]Producer{{MicroPerHour: MicroPerCoin, From: at(0), To: at(10)}}, 2)
	if got.Micro != 50*MicroPerCoin {
		t.Fatalf("contents = %.2f, want 50", coins(got.Micro))
	}
}

func TestNinetyDaysAwayIsBoundedBySiloAndLifespan(t *testing.T) {
	rate := ProducerMicroPerHour(60*60, 108*3600, 1)                // Oak 60 min, 108 h life
	p := []Producer{{MicroPerHour: rate, From: at(0), To: at(108)}} // To is wilts_at, not 90 days
	got := SiloStock{}.Accrue(p, 24)
	maxByLife := rate * 108 // what the plant can ever make
	if float64(got.Micro) > maxByLife+1 {
		t.Fatalf("produced %.1f 🪙, more than its life allows (%.1f)", coins(got.Micro), maxByLife/MicroPerCoin)
	}
	if want := 24 * rate; math.Abs(float64(got.Micro)-want) > 30 { // peak is rounded to an integer, ×24 h
		t.Fatalf("contents = %.2f, Silo (24 h) should hold %.2f", coins(got.Micro), want/MicroPerCoin)
	}
	// the same plant seen after it died produces nothing more
	again := got.Accrue([]Producer{{MicroPerHour: rate, From: at(108), To: at(108)}}, 24)
	if again != got {
		t.Fatal("a plant produced after its lifespan")
	}
}

func TestMaturesAndWiltsInsideTheSameAbsence(t *testing.T) {
	// a plant that only lives from 2 h to 5 h of a 30 h window
	rate := 6.0 * MicroPerCoin
	got := SiloStock{}.Accrue([]Producer{{MicroPerHour: rate, From: at(2), To: at(5)}}, 100)
	if want := 18.0; math.Abs(coins(got.Micro)-want) > 1e-6 {
		t.Fatalf("got %.4f, want %.1f", coins(got.Micro), want)
	}
}

func TestEmptyOrBackwardsWindowsDoNothing(t *testing.T) {
	s := SiloStock{Micro: 7, PeakMicroPerHour: 9}
	for name, p := range map[string][]Producer{
		"no producers":         nil,
		"zero length":          {{MicroPerHour: 1e6, From: at(3), To: at(3)}},
		"clock went backwards": {{MicroPerHour: 1e6, From: at(3), To: at(1)}},
		"zero rate":            {{MicroPerHour: 0, From: at(0), To: at(9)}},
	} {
		if got := s.Accrue(p, 24); got != s {
			t.Errorf("%s: %+v, want unchanged %+v", name, got, s)
		}
	}
}

// One long settlement and many short ones must give the same Silo (the game settles on every request).
func TestSettlingOftenGivesTheSameResultAsOnce(t *testing.T) {
	rnd := rand.New(rand.NewSource(7))
	for trial := 0; trial < 50; trial++ {
		var whole []Producer
		for i := 0; i < 1+rnd.Intn(5); i++ {
			from := rnd.Float64() * 40
			whole = append(whole, Producer{MicroPerHour: (1 + rnd.Float64()*6) * MicroPerCoin, From: at(from), To: at(from + 1 + rnd.Float64()*60)})
		}
		capH := float64(12 + rnd.Intn(60))
		end := 130.0
		once := SiloStock{}.Accrue(clip(whole, 0, end), capH)

		pieces := SiloStock{}
		cuts := []float64{0}
		for c := 0.0; c < end; {
			c += 0.2 + rnd.Float64()*25
			if c > end {
				c = end
			}
			cuts = append(cuts, c)
		}
		for i := 0; i+1 < len(cuts); i++ {
			pieces = pieces.Accrue(clip(whole, cuts[i], cuts[i+1]), capH)
		}
		if diff := math.Abs(float64(once.Micro - pieces.Micro)); diff > 50 { // 50 micro-coins = 0.00005 🪙
			t.Fatalf("trial %d: once %.5f vs pieces %.5f 🪙", trial, coins(once.Micro), coins(pieces.Micro))
		}
		if once.PeakMicroPerHour != pieces.PeakMicroPerHour && math.Abs(float64(once.PeakMicroPerHour-pieces.PeakMicroPerHour)) > 5 {
			t.Fatalf("trial %d: peak %d vs %d", trial, once.PeakMicroPerHour, pieces.PeakMicroPerHour)
		}
	}
}

// clip restricts each producer's interval to [lo, hi] hours, as successive settlements would see them.
func clip(ps []Producer, lo, hi float64) []Producer {
	var out []Producer
	for _, p := range ps {
		f, to := p.From, p.To
		if f.Before(at(lo)) {
			f = at(lo)
		}
		if to.After(at(hi)) {
			to = at(hi)
		}
		if to.After(f) {
			out = append(out, Producer{MicroPerHour: p.MicroPerHour, From: f, To: to})
		}
	}
	return out
}

func TestSiloCapacityHours(t *testing.T) {
	for lvl, want := range map[int]float64{0: 12, 1: 24, 2: 36, 3: 48, 4: 72, 9: 72, -1: 12} {
		if got := SiloCapacityHours(lvl, false); got != want {
			t.Errorf("level %d = %v h, want %v", lvl, got, want)
		}
	}
	if got := SiloCapacityHours(2, true); got != 48 { // GDD §4.5: the Dog adds +12 h
		t.Errorf("level 2 + dog = %v h, want 48", got)
	}
}

func TestProducerRateMatchesGDDTable(t *testing.T) {
	// GDD §4.3: 🪙/h in life: daisy 0.8, tomato 2.9, sunflower 3.5, apple 4.2, oak 4.7
	for _, c := range Crops {
		got := ProducerMicroPerHour(int64(c.DurationMin)*60, int64(c.LifeH)*3600, 1) / MicroPerCoin
		if want := RatePerHour(c.DurationMin, c.LifeH); math.Abs(got-want) > 1e-9 {
			t.Errorf("%s: %.4f vs RatePerHour %.4f", c.Key, got, want)
		}
	}
	if got := ProducerMicroPerHour(600, 24*3600, 2); math.Abs(got/MicroPerCoin-2*0.8333) > 0.001 {
		t.Errorf("multiplier not applied: %.4f", got/MicroPerCoin)
	}
}
