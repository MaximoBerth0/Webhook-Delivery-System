package event

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

type CreateEventRequest struct {
	Type    string
	Payload json.RawMessage
}

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
		CreatedAt: time.Now().UTC(),
	}

	return s.repo.Create(ctx, event)
}

func (s *Service) GetEvent(ctx context.Context, id string) (*Event, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) ListEvents(ctx context.Context, limit int) ([]Event, error) {
	return s.repo.List(ctx, limit)
}
