package attempt

import (
	"context"
	"time"
)

type DeliveryAttemptRepository interface {
	CreateAttempt(ctx context.Context, a *DeliveryAttempt) error
	GetAttempts(ctx context.Context, deliveryID string) ([]DeliveryAttempt, error)

	GetReadyAttempts(ctx context.Context, before time.Time) ([]*DeliveryAttempt, error)

	UpdateStatus(ctx context.Context, attempt *DeliveryAttempt) error
}
