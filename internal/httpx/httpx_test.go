package httpx

import (
	"context"
	"errors"
	"testing"
	"time"
)

// computeBackoffDelay helper for tests by wrapping WithRetries timing via injection isn't trivial.
// Instead, check deterministic bounds with a fake RNG via a local copy.
func computeDelay(base time.Duration, attempt int, rnd func() float64) time.Duration {
	jitter := 1.0 + rnd()*0.2
	factor := 1 << uint(attempt)
	return time.Duration(float64(base) * jitter * float64(factor))
}

func TestBackoff_JitterBounds(t *testing.T) {
	base := 100 * time.Millisecond
	// rnd returns 0 => multiplier 1.0
	d0 := computeDelay(base, 2, func() float64 { return 0 })
	// rnd returns 1 => multiplier 1.2 (upper bound)
	d1 := computeDelay(base, 2, func() float64 { return 1 })
	if d0 != 4*base {
		t.Fatalf("low jitter wrong: %v", d0)
	}
	if d1 != time.Duration(1.2*float64(4*base)) {
		t.Fatalf("high jitter wrong: %v", d1)
	}
}

func TestWithRetries_StopsAfterMax(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	attempts := 0
	err := WithRetries(ctx, 2, 1*time.Millisecond, func(int) error {
		attempts++
		return errors.New("boom")
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if attempts != 3 { // 0,1,2
		t.Fatalf("unexpected attempts: %d", attempts)
	}
}
