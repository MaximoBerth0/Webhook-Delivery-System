package webhook

import (
	"context"
	"errors"
	"net/url"
	"webhook-delivery-system/internal/event"
)

type Service struct {
	repo WebhookRepository
}

func NewService(repo WebhookRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateWebhook(ctx context.Context, webhook *Webhook) error {

	if webhook == nil {
		return errors.New("webhook required")
	}

	if webhook.TargetURL == "" {
		return errors.New("url required")
	}

	if _, err := url.ParseRequestURI(webhook.TargetURL); err != nil {
		return errors.New("invalid webhook url")
	}

	if len(webhook.SubscribedEvents) == 0 {
		return errors.New("at least one event required")
	}

	for _, e := range webhook.SubscribedEvents {
		if !event.IsValidEvent(e) {
			return errors.New("invalid event type")
		}
	}

	return s.repo.Create(ctx, webhook)
}

func (s *Service) GetWebhook(ctx context.Context, id string) (*Webhook, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) ListWebhooksByEvent(ctx context.Context, event event.SubscribedEvent) ([]Webhook, error) {
	return s.repo.ListByEvent(ctx, event)
}

func (s *Service) UpdateWebhook(ctx context.Context, webhook *Webhook) error {

	if webhook == nil {
		return errors.New("webhook required")
	}

	for _, e := range webhook.SubscribedEvents {
		if !event.IsValidEvent(e) {
			return errors.New("invalid event type")
		}
	}

	return s.repo.Update(ctx, webhook)
}

func (s *Service) DeleteWebhook(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
