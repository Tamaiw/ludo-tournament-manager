package ports

import "time"

// Clock is the port over time.Now for testability.
type Clock interface {
	Now() time.Time
}

// RealClock is the production implementation.
type RealClock struct{}

func (RealClock) Now() time.Time { return time.Now().UTC() }
