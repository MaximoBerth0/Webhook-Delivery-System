package attempt

import (
	"context"
	"errors"
	"time"
)

var (
	ErrInvalidDeliveryID = errors.New("invalid delivery id")
)

type Service struct {
	repo DeliveryAttemptRepository
}

func NewService(repo DeliveryAttemptRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateAttempt(ctx context.Context, deliveryID string) error {
	if deliveryID == "" {
		return ErrInvalidDeliveryID
	}

	attempt := &DeliveryAttempt{
		DeliveryID: deliveryID,
		Status:     "pending",
		CreatedAt:  time.Now(),
	}

	return s.repo.Create(ctx, attempt)
}

func (s *Service) GetAttempts(ctx context.Context, deliveryID string) ([]DeliveryAttempt, error) {
	if deliveryID == "" {
		return nil, ErrInvalidDeliveryID
	}

	return s.repo.GetByDeliveryID(ctx, deliveryID)
}
