package clockutil

import (
	"testing"
	"time"
)

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time {
	return c.now
}

func TestSystemNowReturnsUTC(t *testing.T) {
	t.Parallel()

	now := System{}.Now()
	if now.IsZero() {
		t.Fatal("System.Now returned zero time")
	}
	if now.Location() != time.UTC {
		t.Fatalf("System.Now location = %v, want UTC", now.Location())
	}
}

func TestNowUTCAndTimestampAdvancement(t *testing.T) {
	t.Parallel()

	zone := time.FixedZone("test", -7*60*60)
	candidate := time.Date(2026, 3, 25, 12, 0, 0, 0, zone)
	normalized := NowUTC(fixedClock{now: candidate})
	if !normalized.Equal(candidate) || normalized.Location() != time.UTC {
		t.Fatalf(
			"NowUTC(...) = %v (%v), want equivalent UTC time",
			normalized,
			normalized.Location(),
		)
	}

	previous := normalized
	if got := Next(fixedClock{now: previous}, previous); !got.Equal(previous.Add(time.Nanosecond)) {
		t.Fatalf("Next(...) = %v, want one nanosecond after %v", got, previous)
	}

	later := candidate.Add(time.Second)
	if got := Advance(later, previous); !got.Equal(later) || got.Location() != time.UTC {
		t.Fatalf("Advance(...) = %v (%v), want later UTC timestamp", got, got.Location())
	}
}
