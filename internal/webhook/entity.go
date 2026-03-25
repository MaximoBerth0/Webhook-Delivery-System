package webhook

import (
	"errors"
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

func NewWebhook(id, targetURL string, maxAttempts int) (*Webhook, error) {
	if targetURL == "" {
		return nil, errors.New("target URL is required")
	}

	if maxAttempts < 1 {
		return nil, errors.New("max attempts must be >= 1")
	}

	return &Webhook{
		ID:          id,
		TargetURL:   targetURL,
		MaxAttempts: maxAttempts,
		Active:      true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}, nil
}
