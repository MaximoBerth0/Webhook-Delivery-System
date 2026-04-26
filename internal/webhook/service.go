package webhook

import (
	"context"
	"errors"
	"log/slog"

	"webhook-delivery-system/internal/event"
	"webhook-delivery-system/internal/infrastructure"
)

type CreateWebhookRequest struct {
	TargetURL        string
	Secret           string
	MaxAttempts      int
	SubscribedEvents []event.SubscribedEvent
}

type Service struct {
	repo   WebhookRepository
	idGen  infrastructure.Generator
	logger *slog.Logger
}

func NewService(repo WebhookRepository, idGen infrastructure.Generator, logger *slog.Logger) *Service {
	return &Service{repo: repo, idGen: idGen, logger: logger}
}

func (s *Service) CreateWebhook(ctx context.Context, req CreateWebhookRequest) (*Webhook, error) {
	s.logger.Info("creating webhook", slog.String("target_url", req.TargetURL))

	w, err := NewWebhook(req.TargetURL, req.Secret, req.MaxAttempts, req.SubscribedEvents)
	if err != nil {
		if errors.Is(err, ErrURLrequired) || errors.Is(err, ErrInvalidTargetURL) {
			s.logger.Error("invalid target url", slog.String("target_url", req.TargetURL))
		} else if errors.Is(err, ErrSecretRequired) {
			s.logger.Error("secret is required")
		} else if errors.Is(err, ErrInvalidMaxAttempts) {
			s.logger.Error("invalid max attempts", slog.Int("max_attempts", req.MaxAttempts))
		} else if errors.Is(err, ErrNumberOfEvents) || errors.Is(err, ErrInvalidEventType) {
			s.logger.Error("invalid subscribed events")
		}

		return nil, err
	}

	w.ID = s.idGen.Generate()

	if err := s.repo.Create(ctx, w); err != nil {
		s.logger.Error("failed to persist webhook",
			slog.String("webhook_id", w.ID),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	s.logger.Info("webhook created",
		slog.String("webhook_id", w.ID),
		slog.String("target_url", w.TargetURL),
	)

	return w, nil
}

func (s *Service) UpdateWebhook(ctx context.Context, id string, req CreateWebhookRequest) error {
	s.logger.Info("updating webhook", slog.String("webhook_id", id))

	w, err := NewWebhook(req.TargetURL, req.Secret, req.MaxAttempts, req.SubscribedEvents)
	if err != nil {
		s.logger.Error("invalid webhook data",
			slog.String("webhook_id", id),
			slog.String("error", err.Error()),
		)
		return err
	}

	w.ID = id

	if err := s.repo.Update(ctx, w); err != nil {
		s.logger.Error("failed to update webhook",
			slog.String("webhook_id", id),
			slog.String("error", err.Error()),
		)
		return ErrUpdateWebhook
	}

	s.logger.Info("webhook updated", slog.String("webhook_id", id))

	return nil
}

func (s *Service) GetWebhook(ctx context.Context, id string) (*Webhook, error) {
	w, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("failed to get webhook",
			slog.String("webhook_id", id),
			slog.String("error", err.Error()),
		)
		return nil, ErrGetWebhook
	}

	return w, nil
}

func (s *Service) ListWebhooksByEvent(ctx context.Context, e event.SubscribedEvent) ([]Webhook, error) {
	webhooks, err := s.repo.ListByEvent(ctx, e)
	if err != nil {
		s.logger.Error("failed to list webhooks",
			slog.String("event_type", string(e)),
			slog.String("error", err.Error()),
		)
		return nil, ErrListWebhooks
	}

	return webhooks, nil
}

func (s *Service) DeleteWebhook(ctx context.Context, id string) error {
	s.logger.Info("deleting webhook", slog.String("webhook_id", id))

	if err := s.repo.Delete(ctx, id); err != nil {
		s.logger.Error("failed to delete webhook",
			slog.String("webhook_id", id),
			slog.String("error", err.Error()),
		)
		return ErrDeleteWebhook
	}

	s.logger.Info("webhook deleted", slog.String("webhook_id", id))

	return nil
}
