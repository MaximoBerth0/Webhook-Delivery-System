package delivery

import (
	"context"
)

type DeliveryAttemptRepository interface {
	Create(ctx context.Context, a *DeliveryAttempt) error
	GetByDeliveryID(ctx context.Context, deliveryID string) ([]DeliveryAttempt, error)
}
