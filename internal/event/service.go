package event

import (
	"context"
	"errors"
	"log/slog"

	"webhook-delivery-system/internal/infrastructure"
)

type CreateEventRequest struct {
	Type    SubscribedEvent
	Payload []byte
}

type Service struct {
	repo   EventRepository
	idGen  infrastructure.Generator
	logger *slog.Logger
}

func NewService(repo EventRepository, idGen infrastructure.Generator, logger *slog.Logger) *Service {
	return &Service{repo: repo, idGen: idGen, logger: logger}
}

func (s *Service) CreateEvent(ctx context.Context, req CreateEventRequest) (*Event, error) {
	s.logger.Info("creating event", slog.String("event_type", string(req.Type)))

	event, err := NewEvent(req.Type, req.Payload)
	if err != nil {
		if errors.Is(err, ErrInvalidEventType) {
			s.logger.Error("invalid event type",
				slog.String("type", string(req.Type)),
			)
		} else if errors.Is(err, ErrInvalidPayload) {
			s.logger.Error("invalid payload",
				slog.String("type", string(req.Type)),
			)
		}

		return nil, err
	}

	event.ID = s.idGen.Generate()

	if err := s.repo.Create(ctx, event); err != nil {
		s.logger.Error("failed to persist event",
			slog.String("event_id", event.ID),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	s.logger.Info("event created",
		slog.String("event_id", event.ID),
		slog.String("event_type", string(event.Type)),
	)

	return event, nil
}

func (s *Service) GetEvent(ctx context.Context, id string) (*Event, error) {
	event, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("failed to get event", slog.String("event_id", id), slog.String("error", err.Error()))
		return nil, ErrGetEvent
	}

	return event, nil
}

func (s *Service) ListEvents(ctx context.Context, limit int) ([]Event, error) {
	events, err := s.repo.List(ctx, limit)
	if err != nil {
		s.logger.Error("failed to list events", slog.String("error", err.Error()))
		return nil, ErrListEvents
	}

	return events, nil
}
