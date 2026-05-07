package delivery

import (
	"context"
)

type DeliveryRepository interface {
	Create(ctx context.Context, d *Delivery) error

	GetByID(ctx context.Context, id string) (*Delivery, error)
	GetByWebhookID(ctx context.Context, webhookID string) ([]Delivery, error)

	GetPending(ctx context.Context, limit int) ([]Delivery, error)

	UpdateStatus(ctx context.Context, id string, status Status) (*Delivery, error)
}
