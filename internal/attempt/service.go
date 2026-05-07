package attempt

import (
	"context"
	"log/slog"
	"time"
	"webhook-delivery-system/internal/infrastructure"
)

type Service struct {
	repo    DeliveryAttemptRepository
	idGen   infrastructure.Generator
	logger  *slog.Logger
	backoff *BackoffStrategy
}

func NewService(repo DeliveryAttemptRepository, idGen infrastructure.Generator, logger *slog.Logger) *Service {
	return &Service{
		repo:    repo,
		idGen:   idGen,
		logger:  logger,
		backoff: NewDefaultBackoffStrategy(),
	}
}

// private method
func (s *Service) createAttempt(ctx context.Context, deliveryID string, scheduledFor time.Time) (*DeliveryAttempt, error) {
	existing, err := s.repo.GetAttempts(ctx, deliveryID)
	if err != nil {
		s.logger.Error("failed to fetch existing attempts",
			slog.String("delivery_id", deliveryID),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	attemptNumber := len(existing) + 1
	attempt, err := NewDeliveryAttempt(deliveryID, attemptNumber, scheduledFor)
	if err != nil {
		s.logger.Error("failed to create attempt entity",
			slog.String("delivery_id", deliveryID),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	attempt.ID = s.idGen.Generate()

	if err := s.repo.CreateAttempt(ctx, attempt); err != nil {
		s.logger.Error("failed to persist attempt",
			slog.String("delivery_id", deliveryID),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	return attempt, nil
}

// public method
func (s *Service) CreateAttempt(ctx context.Context, deliveryID string) (*DeliveryAttempt, error) {
	s.logger.Info("creating delivery attempt",
		slog.String("delivery_id", deliveryID),
	)

	attempt, err := s.createAttempt(ctx, deliveryID, time.Now())
	if err != nil {
		return nil, err
	}

	s.logger.Info("delivery attempt created successfully",
		slog.String("delivery_id", deliveryID),
		slog.String("attempt_id", attempt.ID),
	)

	return attempt, nil
}

func (s *Service) CreateScheduledAttempt(ctx context.Context, deliveryID string, scheduledFor time.Time) error {
	s.logger.Info("creating scheduled delivery attempt",
		slog.String("delivery_id", deliveryID),
		slog.Time("scheduled_for", scheduledFor),
	)

	attempt, err := s.createAttempt(ctx, deliveryID, scheduledFor)
	if err != nil {
		return err
	}

	s.logger.Info("scheduled delivery attempt created successfully",
		slog.String("delivery_id", deliveryID),
		slog.String("attempt_id", attempt.ID),
		slog.Time("scheduled_for", scheduledFor),
	)

	return nil
}

func (s *Service) GetAttempts(ctx context.Context, deliveryID string) ([]DeliveryAttempt, error) {
	s.logger.Info("fetching delivery attempts",
		slog.String("delivery_id", deliveryID),
	)

	attempts, err := s.repo.GetAttempts(ctx, deliveryID)
	if err != nil {
		s.logger.Error("failed to fetch attempts",
			slog.String("delivery_id", deliveryID),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	s.logger.Info("delivery attempts fetched successfully",
		slog.String("delivery_id", deliveryID),
		slog.Int("count", len(attempts)),
	)

	return attempts, nil
}

func (s *Service) GetReadyAttempts(ctx context.Context) ([]*DeliveryAttempt, error) {
	return s.repo.GetReadyAttempts(ctx, time.Now())
}

func (s *Service) UpdateAttemptStatus(ctx context.Context, attempt *DeliveryAttempt) error {
	return s.repo.UpdateStatus(ctx, attempt)
}

func (s *Service) CalculateNextAttemptTime(attemptNumber int, failedAt time.Time) time.Time {
	return s.backoff.CalculateNextAttemptTime(attemptNumber, failedAt)
}
