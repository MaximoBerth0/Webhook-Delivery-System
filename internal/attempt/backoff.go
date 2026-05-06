package attempt

import (
	"math"
	"time"
)

type BackoffStrategy struct {
	BaseDelay  time.Duration
	MaxDelay   time.Duration
	Multiplier float64
}

func NewDefaultBackoffStrategy() *BackoffStrategy {
	return &BackoffStrategy{
		BaseDelay:  1 * time.Second,
		MaxDelay:   1 * time.Hour,
		Multiplier: 2.0,
	}
}

func (b *BackoffStrategy) CalculateDelay(attemptNumber int) time.Duration {
	if attemptNumber <= 1 {
		return 0
	}

	// attemptNumber 2 -> 2^0 = 1s
	// attemptNumber 3 -> 2^1 = 2s
	// attemptNumber 4 -> 2^2 = 4s
	exponent := float64(attemptNumber - 2)
	delayF := float64(b.BaseDelay) * math.Pow(b.Multiplier, exponent)

	if delayF >= float64(b.MaxDelay) {
		return b.MaxDelay
	}
	return time.Duration(delayF)
}

func (b *BackoffStrategy) CalculateNextAttemptTime(attemptNumber int, failedAt time.Time) time.Time {
	delay := b.CalculateDelay(attemptNumber + 1)
	return failedAt.Add(delay)
}
