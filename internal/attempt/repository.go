package attempt

import (
	"context"
	"time"
)

type DeliveryAttemptRepository interface {
	Create(ctx context.Context, a *DeliveryAttempt) error
	GetByDeliveryID(ctx context.Context, deliveryID string) ([]DeliveryAttempt, error)

	GetReadyAttempts(ctx context.Context, before time.Time) ([]*DeliveryAttempt, error)
}
