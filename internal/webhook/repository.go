package webhook

import (
	"context"
	"webhook-delivery-system/internal/event"
)

type Repository interface {
	Create(ctx context.Context, webhook *Webhook) error
	GetByID(ctx context.Context, id string) (*Webhook, error)
	ListByEvent(ctx context.Context, event event.SubscribedEvent) ([]Webhook, error)
	Update(ctx context.Context, webhook *Webhook) error
	Delete(ctx context.Context, id string) error
}
