package webhook

import (
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
		return nil, ErrURLrequired
	}
	if _, err := url.ParseRequestURI(targetURL); err != nil {
		return nil, ErrInvalidTargetURL
	}
	if secret == "" {
		return nil, ErrSecretRequired
	}
	if maxAttempts < 1 {
		return nil, ErrInvalidMaxAttempts
	}
	if len(events) == 0 {
		return nil, ErrNumberOfEvents
	}
	for _, e := range events {
		if !event.IsValidEvent(e) {
			return nil, ErrInvalidEventType
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
