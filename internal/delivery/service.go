package delivery

import (
	"context"
)

type Service struct {
	repo DeliveryRepository
}

func NewService(repo DeliveryRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, eventID, webhookID string) (*Delivery, error) {
	d, err := NewDelivery(eventID, webhookID)
	if err != nil {
		return nil, err // add domain error
	}

	if err := s.repo.Create(ctx, d); err != nil {
		return nil, ErrCreateDelivery
	}

	return d, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (*Delivery, error) {
	if id == "" {
		return nil, ErrInvalidDeliveryID
	}
	d, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrDeliveryNotFound
	}

	return d, nil
}

func (s *Service) GetPending(ctx context.Context, limit int) ([]Delivery, error) {
	deliveries, err := s.repo.GetPending(ctx, limit)
	if err != nil {
		return nil, ErrGetPending
	}

	return deliveries, nil
}

func (s *Service) GetRetryable(ctx context.Context, limit int) ([]Delivery, error) {
	deliveries, err := s.repo.GetRetryable(ctx, limit)
	if err != nil {
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
		return nil, ErrUpdateStatus
	}

	return d, nil
}

func (s *Service) IncrementAttempts(ctx context.Context, id string) (*Delivery, error) {
	if id == "" {
		return nil, ErrInvalidDeliveryID
	}
	d, err := s.repo.IncrementAttempts(ctx, id)
	if err != nil {
		return nil, ErrIncrementAttempts
	}

	return d, nil
}
