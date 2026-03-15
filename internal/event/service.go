package event

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

type Service struct {
	repo EventRepository
}

func NewService(repo EventRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateEvent(ctx context.Context, req CreateEventRequest) error {

	if req.Type == "" {
		return errors.New("event type required")
	}
	event := &Event{
		ID:        uuid.New().String(),
		Type:      SubscribedEvent(req.Type),
		Payload:   req.Payload,
		CreatedAt: time.Now(),
	}

	return s.repo.Create(ctx, event)
}

func (s *Service) GetEvent(ctx context.Context, id string) (*Event, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) ListEvents(ctx context.Context, limit int) ([]Event, error) {
	return s.repo.List(ctx, limit)
}
