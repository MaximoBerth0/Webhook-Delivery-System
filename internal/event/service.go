package event

import (
	"context"
	"fmt"
)

type CreateEventRequest struct {
	Type    SubscribedEvent
	Payload []byte
}

type Service struct {
	repo EventRepository
}

func NewService(repo EventRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateEvent(ctx context.Context, req CreateEventRequest) (*Event, error) {
	event, err := NewEvent(req.Type, req.Payload)
	if err != nil {
		return nil, fmt.Errorf("creating event: %w", err)
	}
	if err := s.repo.Create(ctx, event); err != nil {
		return nil, err
	}
	return event, nil
}

func (s *Service) GetEvent(ctx context.Context, id string) (*Event, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) ListEvents(ctx context.Context, limit int) ([]Event, error) {
	return s.repo.List(ctx, limit)
}
