package webhook

import (
	"errors"
	"fmt"
	"net/url"
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

func NewWebhook(targetURL, secret string, maxAttempts int, events []event.SubscribedEvent) (*Webhook, error) {
	if targetURL == "" {
		return nil, errors.New("target URL is required")
	}
	if _, err := url.ParseRequestURI(targetURL); err != nil {
		return nil, errors.New("invalid target URL")
	}
	if secret == "" {
		return nil, errors.New("secret is required")
	}
	if maxAttempts < 1 {
		return nil, errors.New("max attempts must be >= 1")
	}
	if len(events) == 0 {
		return nil, errors.New("at least one event required")
	}
	for _, e := range events {
		if !event.IsValidEvent(e) {
			return nil, fmt.Errorf("invalid event type: %s", e)
		}
	}
	return &Webhook{
		TargetURL:        targetURL,
		Secret:           secret,
		MaxAttempts:      maxAttempts,
		SubscribedEvents: events,
		Active:           true,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}, nil
}
