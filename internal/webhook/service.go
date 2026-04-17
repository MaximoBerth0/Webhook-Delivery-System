package webhook

import (
	"context"
	"fmt"
	"webhook-delivery-system/internal/event"
)

type Service struct {
	repo WebhookRepository
}

func NewService(repo WebhookRepository) *Service {
	return &Service{repo: repo}
}

type CreateWebhookRequest struct {
	TargetURL        string
	Secret           string
	MaxAttempts      int
	SubscribedEvents []event.SubscribedEvent
}

func (s *Service) CreateWebhook(ctx context.Context, req CreateWebhookRequest) (*Webhook, error) {
	w, err := NewWebhook(req.TargetURL, req.Secret, req.MaxAttempts, req.SubscribedEvents)
	if err != nil {
		return nil, fmt.Errorf("creating webhook: %w", err)
	}
	if err := s.repo.Create(ctx, w); err != nil {
		return nil, err
	}
	return w, nil
}

func (s *Service) UpdateWebhook(ctx context.Context, id string, req CreateWebhookRequest) error {
	w, err := NewWebhook(req.TargetURL, req.Secret, req.MaxAttempts, req.SubscribedEvents)
	if err != nil {
		return fmt.Errorf("updating webhook: %w", err)
	}
	w.ID = id
	return s.repo.Update(ctx, w)
}

func (s *Service) GetWebhook(ctx context.Context, id string) (*Webhook, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) ListWebhooksByEvent(ctx context.Context, e event.SubscribedEvent) ([]Webhook, error) {
	return s.repo.ListByEvent(ctx, e)
}

func (s *Service) DeleteWebhook(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
