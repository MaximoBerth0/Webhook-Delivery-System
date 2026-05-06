package attempt

import (
	"testing"
	"time"
)

// run the test:
// go test ./internal/attempt/ -v 2>&1

func TestCalculateDelay(t *testing.T) {
	b := NewDefaultBackoffStrategy()

	tests := []struct {
		attemptNumber int
		expected      time.Duration
	}{
		{1, 0},                  // first attempt: no delay
		{2, 1 * time.Second},    // 2^0 * 1s = 1s
		{3, 2 * time.Second},    // 2^1 * 1s = 2s
		{4, 4 * time.Second},    // 2^2 * 1s = 4s
		{5, 8 * time.Second},    // 2^3 * 1s = 8s
		{10, 256 * time.Second}, // 2^8 * 1s = 256s
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := b.CalculateDelay(tt.attemptNumber)
			if got != tt.expected {
				t.Errorf("CalculateDelay(%d) = %v, want %v", tt.attemptNumber, got, tt.expected)
			}
		})
	}
}

func TestCalculateDelay_AttemptZeroOrNegative(t *testing.T) {
	b := NewDefaultBackoffStrategy()

	for _, n := range []int{0, -1, -100} {
		got := b.CalculateDelay(n)
		if got != 0 {
			t.Errorf("CalculateDelay(%d) = %v, want 0", n, got)
		}
	}
}

func TestCalculateDelay_CapsAtMaxDelay(t *testing.T) {
	b := NewDefaultBackoffStrategy()

	// attempt 63+ would overflow without the cap, verify it never exceeds MaxDelay
	for _, n := range []int{40, 63, 100} {
		got := b.CalculateDelay(n)
		if got != b.MaxDelay {
			t.Errorf("CalculateDelay(%d) = %v, want MaxDelay %v", n, got, b.MaxDelay)
		}
	}
}

func TestCalculateDelay_CustomStrategy(t *testing.T) {
	b := &BackoffStrategy{
		BaseDelay:  500 * time.Millisecond,
		MaxDelay:   10 * time.Second,
		Multiplier: 3.0,
	}

	// attempt 2: 3^0 * 500ms = 500ms
	// attempt 3: 3^1 * 500ms = 1500ms
	// attempt 4: 3^2 * 500ms = 4500ms
	// attempt 5: 3^3 * 500ms = 13500ms -> capped at 10s
	cases := []struct {
		n    int
		want time.Duration
	}{
		{2, 500 * time.Millisecond},
		{3, 1500 * time.Millisecond},
		{4, 4500 * time.Millisecond},
		{5, 10 * time.Second},
	}

	for _, c := range cases {
		got := b.CalculateDelay(c.n)
		if got != c.want {
			t.Errorf("CalculateDelay(%d) = %v, want %v", c.n, got, c.want)
		}
	}
}

func TestCalculateNextAttemptTime(t *testing.T) {
	b := NewDefaultBackoffStrategy()
	base := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	// CalculateNextAttemptTime(n, t) calls CalculateDelay(n+1)
	// attempt 1 failed -> next is attempt 2 -> delay = CalculateDelay(2) = 1s
	// attempt 2 failed -> next is attempt 3 -> delay = CalculateDelay(3) = 2s
	cases := []struct {
		currentAttempt int
		wantDelay      time.Duration
	}{
		{1, 1 * time.Second},
		{2, 2 * time.Second},
		{3, 4 * time.Second},
	}

	for _, c := range cases {
		got := b.CalculateNextAttemptTime(c.currentAttempt, base)
		want := base.Add(c.wantDelay)
		if !got.Equal(want) {
			t.Errorf("CalculateNextAttemptTime(%d) = %v, want %v", c.currentAttempt, got, want)
		}
	}
}

func TestNewDefaultBackoffStrategy(t *testing.T) {
	b := NewDefaultBackoffStrategy()

	if b.BaseDelay != 1*time.Second {
		t.Errorf("BaseDelay = %v, want 1s", b.BaseDelay)
	}
	if b.MaxDelay != 1*time.Hour {
		t.Errorf("MaxDelay = %v, want 1h", b.MaxDelay)
	}
	if b.Multiplier != 2.0 {
		t.Errorf("Multiplier = %v, want 2.0", b.Multiplier)
	}
}
