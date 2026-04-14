package attempt

import "context"

type Service struct {
	repo DeliveryAttemptRepository
}

func NewService(repo DeliveryAttemptRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateAttempt(ctx context.Context, deliveryID string) error {
	attempt, err := NewDeliveryAttempt(deliveryID) // 👈 validation happens here
	if err != nil {
		return err
	}
	return s.repo.Create(ctx, attempt)
}

func (s *Service) GetAttempts(ctx context.Context, deliveryID string) ([]DeliveryAttempt, error) {
	return s.repo.GetByDeliveryID(ctx, deliveryID)
}
