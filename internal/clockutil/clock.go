package clockutil

import "time"

// Clock supplies the current time to services that create persisted
// timestamps.
type Clock interface {
	Now() time.Time
}

// System reads the current time from the system wall clock.
type System struct{}

// Now returns the current system time normalized to UTC.
func (System) Now() time.Time {
	return time.Now().UTC()
}

// NowUTC reads a Clock and normalizes its result to UTC.
func NowUTC(value Clock) time.Time {
	return value.Now().UTC()
}

// Next reads a Clock and returns a timestamp strictly later than previous.
func Next(value Clock, previous time.Time) time.Time {
	return Advance(NowUTC(value), previous)
}

// Advance returns candidate normalized to UTC when it is later than previous.
// When wall-clock time has not advanced, it advances previous by one
// nanosecond to preserve persisted timestamp ordering.
func Advance(candidate, previous time.Time) time.Time {
	candidate = candidate.UTC()
	if candidate.After(previous) {
		return candidate
	}
	return previous.Add(time.Nanosecond)
}
