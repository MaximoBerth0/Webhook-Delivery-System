package delivery

import (
	"context"
	"log/slog"

	"webhook-delivery-system/internal/infrastructure"
)

type Service struct {
	repo   DeliveryRepository
	idGen  infrastructure.Generator
	logger *slog.Logger
}

func NewService(repo DeliveryRepository, idGen infrastructure.Generator, logger *slog.Logger) *Service {
	return &Service{repo: repo, idGen: idGen, logger: logger}
}

func (s *Service) Create(ctx context.Context, eventID, webhookID string) (*Delivery, error) {
	s.logger.Info("creating delivery", slog.String("event_id", eventID), slog.String("webhook_id", webhookID))

	d, err := NewDelivery(eventID, webhookID)
	if err != nil {
		s.logger.Error("invalid delivery params", slog.String("error", err.Error()))
		return nil, err
	}
	d.ID = s.idGen.Generate()

	if err := s.repo.Create(ctx, d); err != nil {
		s.logger.Error("failed to persist delivery", slog.String("delivery_id", d.ID), slog.String("error", err.Error()))
		return nil, ErrCreateDelivery
	}

	s.logger.Info("delivery created", slog.String("delivery_id", d.ID))
	return d, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (*Delivery, error) {
	if id == "" {
		return nil, ErrInvalidDeliveryID
	}
	d, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("delivery not found", slog.String("delivery_id", id), slog.String("error", err.Error()))
		return nil, ErrDeliveryNotFound
	}

	return d, nil
}

func (s *Service) ListByWebhook(ctx context.Context, webhookID string) ([]Delivery, error) {
	return s.repo.GetByWebhookID(ctx, webhookID)
}

func (s *Service) GetPending(ctx context.Context, limit int) ([]Delivery, error) {
	deliveries, err := s.repo.GetPending(ctx, limit)
	if err != nil {
		s.logger.Error("failed to get pending deliveries", slog.String("error", err.Error()))
		return nil, ErrGetPending
	}

	return deliveries, nil
}

func (s *Service) GetRetryable(ctx context.Context, limit int) ([]Delivery, error) {
	deliveries, err := s.repo.GetRetryable(ctx, limit)
	if err != nil {
		s.logger.Error("failed to get retryable deliveries", slog.String("error", err.Error()))
		return nil, ErrGetRetryable
	}

	return deliveries, nil
}

func (s *Service) UpdateStatus(ctx context.Context, id string, status Status) (*Delivery, error) {
	if id == "" {
		return nil, ErrInvalidDeliveryID
	}
	d, err := s.repo.UpdateStatus(ctx, id, status)
	if err != nil {
		s.logger.Error("failed to update delivery status", slog.String("delivery_id", id), slog.String("status", string(status)), slog.String("error", err.Error()))
		return nil, ErrUpdateStatus
	}

	s.logger.Info("delivery status updated", slog.String("delivery_id", id), slog.String("status", string(status)))
	return d, nil
}

func (s *Service) IncrementAttempts(ctx context.Context, id string) (*Delivery, error) {
	if id == "" {
		return nil, ErrInvalidDeliveryID
	}
	d, err := s.repo.IncrementAttempts(ctx, id)
	if err != nil {
		s.logger.Error("failed to increment attempts", slog.String("delivery_id", id), slog.String("error", err.Error()))
		return nil, ErrIncrementAttempts
	}

	return d, nil
}
