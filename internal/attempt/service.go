package attempt

import (
	"context"
	"log/slog"

	"webhook-delivery-system/internal/infrastructure"
)

type Service struct {
	repo   DeliveryAttemptRepository
	idGen  infrastructure.Generator
	logger *slog.Logger
}

func NewService(repo DeliveryAttemptRepository, idGen infrastructure.Generator, logger *slog.Logger) *Service {
	return &Service{
		repo:   repo,
		idGen:  idGen,
		logger: logger,
	}
}

func (s *Service) CreateAttempt(ctx context.Context, deliveryID string) error {
	s.logger.Info("creating delivery attempt",
		slog.String("delivery_id", deliveryID),
	)

	existing, err := s.repo.GetByDeliveryID(ctx, deliveryID)
	if err != nil {
		s.logger.Error("failed to fetch existing attempts",
			slog.String("delivery_id", deliveryID),
			slog.String("error", err.Error()),
		)
		return err
	}
	attemptNumber := len(existing) + 1

	attempt, err := NewDeliveryAttempt(deliveryID, attemptNumber)
	if err != nil {
		s.logger.Error("failed to create attempt entity",
			slog.String("delivery_id", deliveryID),
			slog.String("error", err.Error()),
		)
		return err
	}
	attempt.ID = s.idGen.Generate()

	if err := s.repo.Create(ctx, attempt); err != nil {
		s.logger.Error("failed to persist attempt",
			slog.String("delivery_id", deliveryID),
			slog.String("error", err.Error()),
		)
		return err
	}

	s.logger.Info("delivery attempt created successfully",
		slog.String("delivery_id", deliveryID),
		slog.String("attempt_id", attempt.ID),
	)

	return nil
}

func (s *Service) GetAttempts(ctx context.Context, deliveryID string) ([]DeliveryAttempt, error) {
	s.logger.Info("fetching delivery attempts",
		slog.String("delivery_id", deliveryID),
	)

	attempts, err := s.repo.GetByDeliveryID(ctx, deliveryID)
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
