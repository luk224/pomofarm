package game

import (
	"errors"
	"math"
	"time"
)

// Pomodoro statuses (match the pomodoros.status column).
const (
	StatusRunning   = "running"
	StatusPaused    = "paused"
	StatusCompleted = "completed"
	StatusCancelled = "cancelled"
)

var (
	ErrNotRunning = errors.New("pomodoro is not running")
	ErrNotPaused  = errors.New("pomodoro is not paused")
	ErrNotActive  = errors.New("pomodoro is already finished")
	ErrNotDone    = errors.New("pomodoro time has not elapsed yet")
)

// Pomodoro is the timer state persisted by the server (GDD §6.1). Remaining time
// is always derived from these timestamps, never from an incremental counter.
type Pomodoro struct {
	PlannedS     int64 // planned duration in seconds
	StartedAt    time.Time
	PausedAt     *time.Time // non-nil while paused
	PausedTotalS int64      // whole seconds spent paused in previous pauses
	EndedAt      *time.Time
	Status       string
}

// Start begins a running Pomodoro of the given duration.
func Start(now time.Time, plannedS int64) Pomodoro {
	return Pomodoro{PlannedS: plannedS, StartedAt: now, Status: StatusRunning}
}

// Active reports whether the Pomodoro is running or paused.
func (p *Pomodoro) Active() bool { return p.Status == StatusRunning || p.Status == StatusPaused }

// Elapsed is the focused time so far, in seconds (fractional), never negative
// and never above the planned duration. While paused it is frozen at PausedAt.
// A clock that went backwards simply yields 0.
func (p *Pomodoro) Elapsed(now time.Time) float64 {
	ref := now
	switch {
	case p.EndedAt != nil && p.Status == StatusCancelled:
		ref = *p.EndedAt
	case p.PausedAt != nil:
		ref = *p.PausedAt
	}
	e := ref.Sub(p.StartedAt).Seconds() - float64(p.PausedTotalS)
	return math.Min(math.Max(e, 0), float64(p.PlannedS))
}

// Remaining = planned − (now − started − pausedTotal), clamped to [0, planned].
func (p *Pomodoro) Remaining(now time.Time) time.Duration {
	if p.Status == StatusCompleted {
		return 0
	}
	return time.Duration((float64(p.PlannedS) - p.Elapsed(now)) * float64(time.Second))
}

// Pause freezes the timer at now.
func (p *Pomodoro) Pause(now time.Time) error {
	if p.Status != StatusRunning {
		if p.Active() {
			return ErrNotRunning
		}
		return ErrNotActive
	}
	t := laterOf(now, p.StartedAt)
	p.PausedAt = &t
	p.Status = StatusPaused
	return nil
}

// Resume unfreezes the timer, adding the pause length (whole seconds, rounded)
// to PausedTotalS. A resume timestamp before the pause (clock went backwards)
// adds nothing.
func (p *Pomodoro) Resume(now time.Time) error {
	if p.Status != StatusPaused {
		if p.Active() {
			return ErrNotPaused
		}
		return ErrNotActive
	}
	if d := now.Sub(*p.PausedAt).Seconds(); d > 0 {
		p.PausedTotalS += int64(math.Round(d))
	}
	p.PausedAt = nil
	p.Status = StatusRunning
	return nil
}

// Cancel abandons the Pomodoro (a deliberate player action: no reward).
func (p *Pomodoro) Cancel(now time.Time) error {
	if !p.Active() {
		return ErrNotActive
	}
	t := laterOf(now, p.StartedAt)
	p.EndedAt = &t
	p.PausedAt = nil
	p.Status = StatusCancelled
	return nil
}

// Complete finishes the Pomodoro once its time has elapsed. Completing while
// paused is allowed if the time was already up when it was paused.
func (p *Pomodoro) Complete(now time.Time) error {
	if !p.Active() {
		return ErrNotActive
	}
	if p.Remaining(now) > 0 {
		return ErrNotDone
	}
	t := p.FinishedAt()
	p.EndedAt = &t
	p.PausedAt = nil
	p.Status = StatusCompleted
	return nil
}

// FinishedAt is the instant the planned time was (or will be) used up: the
// pause time if paused, otherwise start + planned + total pauses. Recording it
// instead of "now" keeps completion exact even if the server only notices later.
func (p *Pomodoro) FinishedAt() time.Time {
	if p.PausedAt != nil {
		return *p.PausedAt
	}
	return p.StartedAt.Add(time.Duration(p.PlannedS+p.PausedTotalS) * time.Second)
}

// Finished reports whether the planned time has fully elapsed (it may still
// need the player to harvest it).
func (p *Pomodoro) Finished(now time.Time) bool { return p.Active() && p.Remaining(now) <= 0 }

func laterOf(a, b time.Time) time.Time {
	if a.Before(b) {
		return b
	}
	return a
}
