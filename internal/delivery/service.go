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

func (s *Service) Create(ctx context.Context, d *Delivery) error {
    if d.ID == "" {
        return ErrInvalidDeliveryID
    }
    return nil

}