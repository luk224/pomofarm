package game

import "time"

// Clock abstracts "now" so time-dependent rules can be tested. In production
// it is the server clock; the client clock is never used (GDD §6.1).
type Clock interface{ Now() time.Time }

// SystemClock is the real server clock.
type SystemClock struct{}

func (SystemClock) Now() time.Time { return time.Now() }

// FakeClock is a manually advanced clock for tests.
type FakeClock struct{ T time.Time }

func (c *FakeClock) Now() time.Time          { return c.T }
func (c *FakeClock) Advance(d time.Duration) { c.T = c.T.Add(d) }
