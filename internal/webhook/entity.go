package webhook

import (
	"time"
	"webhook-delivery-system/internal/event"
)

type Webhook struct {
	ID               string
	TargetURL        string
	SubscribedEvents []event.SubscribedEvent
	Secret           string
	Active           bool
	MaxAttempts      int
	CreatedAt        time.Time
	UpdatedAt        time.Time
}
