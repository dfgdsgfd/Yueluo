package handlers

import (
	"testing"
	"time"
)

func TestAdminLoginMemoryLimiterLocksIPAfterFiveFailures(t *testing.T) {
	now := time.Date(2026, time.June, 17, 12, 0, 0, 0, time.UTC)
	limiter := newAdminLoginMemoryLimiter()
	limiter.now = func() time.Time { return now }

	for attempt := 1; attempt <= adminLoginMaxFailures; attempt++ {
		state := limiter.failure("ip-a")
		if state.FailedAttempts != attempt {
			t.Fatalf("attempt %d failures = %d", attempt, state.FailedAttempts)
		}
		if state.Locked != (attempt == adminLoginMaxFailures) {
			t.Fatalf("attempt %d locked = %v", attempt, state.Locked)
		}
	}

	state := limiter.status("ip-a")
	if !state.Locked || state.RetryAfter != adminLoginLockDuration {
		t.Fatalf("locked state = %#v", state)
	}
	if other := limiter.status("ip-b"); other.Locked {
		t.Fatalf("different IP should not be locked: %#v", other)
	}
}

func TestAdminLoginMemoryLimiterExpiresAndClears(t *testing.T) {
	now := time.Date(2026, time.June, 17, 12, 0, 0, 0, time.UTC)
	limiter := newAdminLoginMemoryLimiter()
	limiter.now = func() time.Time { return now }

	for range adminLoginMaxFailures {
		limiter.failure("ip-a")
	}
	now = now.Add(adminLoginLockDuration + time.Second)
	if state := limiter.status("ip-a"); state.Locked {
		t.Fatalf("expired IP lock remains active: %#v", state)
	}

	limiter.failure("ip-a")
	limiter.clear("ip-a")
	if state := limiter.status("ip-a"); state.Locked || state.FailedAttempts != 0 {
		t.Fatalf("cleared limiter state = %#v", state)
	}
}

func TestAdminLoginIPKeyDoesNotExposeRawAddress(t *testing.T) {
	const ip = "203.0.113.42"
	key := adminLoginIPKey(ip)
	if key == ip || len(key) != 32 {
		t.Fatalf("adminLoginIPKey(%q) = %q", ip, key)
	}
}
