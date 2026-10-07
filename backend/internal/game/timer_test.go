package game

import (
	"errors"
	"testing"
	"time"
)

var t0 = time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)

func sec(n int) time.Duration { return time.Duration(n) * time.Second }

func within(t *testing.T, got, want, tol time.Duration, msg string) {
	t.Helper()
	d := got - want
	if d < 0 {
		d = -d
	}
	if d > tol {
		t.Fatalf("%s: got %v, want %v ±%v", msg, got, want, tol)
	}
}

func TestRemainingCountsDown(t *testing.T) {
	p := Start(t0, 1500)
	within(t, p.Remaining(t0), sec(1500), 0, "at start")
	within(t, p.Remaining(t0.Add(sec(600))), sec(900), 0, "after 10 min")
	within(t, p.Remaining(t0.Add(sec(1500))), 0, 0, "at end")
	within(t, p.Remaining(t0.Add(sec(99999))), 0, 0, "long after the end never negative")
}

// "Cerrar el navegador y volver": nothing is stored but timestamps, so a client
// that disappears for an hour and comes back sees exactly the right time.
func TestCloseBrowserAndReturnIsExact(t *testing.T) {
	p := Start(t0, 1500)
	within(t, p.Remaining(t0.Add(sec(1))), sec(1499), time.Second, "before closing")
	within(t, p.Remaining(t0.Add(sec(1000))), sec(500), time.Second, "after coming back")
}

func TestPauseFreezesTimer(t *testing.T) {
	p := Start(t0, 1500)
	if err := p.Pause(t0.Add(sec(300))); err != nil {
		t.Fatal(err)
	}
	// however long we wait, remaining stays frozen at 1200 s
	within(t, p.Remaining(t0.Add(sec(300))), sec(1200), 0, "just paused")
	within(t, p.Remaining(t0.Add(48*time.Hour)), sec(1200), 0, "paused for two days")
	if err := p.Resume(t0.Add(48*time.Hour + sec(300))); err != nil {
		t.Fatal(err)
	}
	within(t, p.Remaining(t0.Add(48*time.Hour+sec(300))), sec(1200), time.Second, "right after resume")
	within(t, p.Remaining(t0.Add(48*time.Hour+sec(900))), sec(600), time.Second, "10 min after resume")
	if p.Status != StatusRunning || p.PausedAt != nil {
		t.Fatalf("state after resume: %+v", p)
	}
}

func TestMultiplePausesAccumulate(t *testing.T) {
	p := Start(t0, 1500)
	now := t0
	for i := 0; i < 3; i++ {
		now = now.Add(sec(100))
		if err := p.Pause(now); err != nil {
			t.Fatal(err)
		}
		now = now.Add(sec(50 + 100*i))
		if err := p.Resume(now); err != nil {
			t.Fatal(err)
		}
	}
	// 300 s worked, pauses 50+150+250 = 450 s
	if p.PausedTotalS != 450 {
		t.Fatalf("PausedTotalS = %d, want 450", p.PausedTotalS)
	}
	within(t, p.Remaining(now), sec(1200), time.Second, "after 3 pauses")
}

func TestIllegalTransitions(t *testing.T) {
	p := Start(t0, 600)
	if err := p.Resume(t0); !errors.Is(err, ErrNotPaused) {
		t.Fatalf("resume while running: %v", err)
	}
	p.Pause(t0.Add(sec(10)))
	if err := p.Pause(t0.Add(sec(20))); !errors.Is(err, ErrNotRunning) {
		t.Fatalf("pause while paused: %v", err)
	}
	if err := p.Cancel(t0.Add(sec(30))); err != nil {
		t.Fatal(err)
	}
	for name, err := range map[string]error{
		"pause": p.Pause(t0), "resume": p.Resume(t0), "cancel": p.Cancel(t0), "complete": p.Complete(t0),
	} {
		if !errors.Is(err, ErrNotActive) {
			t.Errorf("%s after cancel: %v, want ErrNotActive", name, err)
		}
	}
}

func TestCompleteOnlyWhenTimeIsUp(t *testing.T) {
	p := Start(t0, 600)
	if err := p.Complete(t0.Add(sec(599))); !errors.Is(err, ErrNotDone) {
		t.Fatalf("complete 1 s early: %v", err)
	}
	if p.Finished(t0.Add(sec(599))) {
		t.Fatal("Finished true 1 s early")
	}
	if !p.Finished(t0.Add(sec(600))) {
		t.Fatal("Finished false at the end")
	}
	if err := p.Complete(t0.Add(sec(600))); err != nil {
		t.Fatal(err)
	}
	if p.Status != StatusCompleted || p.EndedAt == nil || p.Remaining(t0) != 0 {
		t.Fatalf("after complete: %+v", p)
	}
}

func TestPausingCannotGainTime(t *testing.T) {
	p := Start(t0, 600)
	p.Pause(t0.Add(sec(590)))
	if err := p.Complete(t0.Add(sec(5000))); !errors.Is(err, ErrNotDone) {
		t.Fatalf("complete while paused with 10 s left: %v", err)
	}
	p.Resume(t0.Add(sec(5000)))
	if err := p.Complete(t0.Add(sec(5009))); !errors.Is(err, ErrNotDone) {
		t.Fatalf("complete 1 s early after resume: %v", err)
	}
	if err := p.Complete(t0.Add(sec(5010))); err != nil {
		t.Fatalf("complete on time after resume: %v", err)
	}
}

// Reloj del sistema retrocedido: nothing goes negative or beyond the plan.
func TestClockGoingBackwards(t *testing.T) {
	p := Start(t0, 600)
	within(t, p.Remaining(t0.Add(-time.Hour)), sec(600), 0, "now before start")
	p.Pause(t0.Add(sec(100)))
	if err := p.Resume(t0.Add(sec(50))); err != nil { // resume "before" the pause
		t.Fatal(err)
	}
	if p.PausedTotalS != 0 {
		t.Fatalf("PausedTotalS = %d, want 0 (negative pause ignored)", p.PausedTotalS)
	}
	if got := p.Remaining(t0.Add(sec(100))); got > sec(600) || got < 0 {
		t.Fatalf("remaining out of range: %v", got)
	}
	q := Start(t0, 600)
	if err := q.Pause(t0.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	within(t, q.Remaining(t0), sec(600), 0, "paused with a backwards clock")
}

func TestCancelFreezesRemaining(t *testing.T) {
	p := Start(t0, 600)
	p.Cancel(t0.Add(sec(100)))
	within(t, p.Remaining(t0.Add(sec(400))), sec(500), 0, "after cancel")
}

func TestFakeClock(t *testing.T) {
	c := &FakeClock{T: t0}
	var clk Clock = c
	p := Start(clk.Now(), 60)
	c.Advance(sec(30))
	within(t, p.Remaining(clk.Now()), sec(30), 0, "fake clock")
}
