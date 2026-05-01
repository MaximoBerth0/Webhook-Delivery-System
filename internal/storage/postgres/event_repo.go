package postgres

import (
	"context"
	"errors"
	"log/slog"
	"webhook-delivery-system/internal/event"
	"webhook-delivery-system/internal/storage"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/jackc/pgx/v5"
)

type EventRepository struct {
	db     storage.DBTX
	logger *slog.Logger
	tracer trace.Tracer
}

func NewEventRepository(db storage.DBTX, logger *slog.Logger) *EventRepository {
	return &EventRepository{db: db, logger: logger, tracer: otel.Tracer("repository.event")}
}

func (r *EventRepository) Create(ctx context.Context, e *event.Event) error {
	ctx, span := r.tracer.Start(ctx, "EventRepository.Create")
	defer span.End()

	query := `
		INSERT INTO events (type, payload, created_at)
		VALUES ($1, $2, $3)
		RETURNING id
	`
	err := r.db.QueryRow(ctx, query, e.Type, e.Payload, e.CreatedAt).Scan(&e.ID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create event")
		r.logger.ErrorContext(ctx, "failed to create event",
			"error", err,
			"event_type", e.Type,
		)
		return err
	}
	span.SetAttributes(attribute.String("event_id", e.ID))
	span.SetStatus(codes.Ok, "event created successfully")
	return nil
}

func (r *EventRepository) GetByID(ctx context.Context, id string) (*event.Event, error) {
	ctx, span := r.tracer.Start(ctx, "EventRepository.GetByID")
	defer span.End()

	span.SetAttributes(attribute.String("event_id", id))

	query := `SELECT id, type, payload, created_at FROM events WHERE id = $1`

	e := new(event.Event)

	err := r.db.QueryRow(ctx, query, id).
		Scan(&e.ID, &e.Type, &e.Payload, &e.CreatedAt)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to query event")
		r.logger.ErrorContext(ctx, "failed to query event by id",
			"error", err,
			"event_id", id,
		)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, storage.ErrEventNotFound
		}
		return nil, err
	}

	span.SetAttributes(attribute.String("event_type", string(e.Type)))
	span.SetStatus(codes.Ok, "event retrieved successfully")
	return e, nil
}

func (r *EventRepository) List(ctx context.Context, limit int) ([]event.Event, error) {
	ctx, span := r.tracer.Start(ctx, "EventRepository.List")
	defer span.End()

	span.SetAttributes(attribute.Int("limit", limit))

	query := `
		SELECT id, type, payload, created_at
		FROM events
		ORDER BY created_at DESC
		LIMIT $1
	`

	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to query events")
		r.logger.ErrorContext(ctx, "failed to query events",
			"error", err,
			"limit", limit,
		)
		return nil, err
	}
	defer rows.Close()

	var events []event.Event

	for rows.Next() {
		var e event.Event

		err := rows.Scan(
			&e.ID,
			&e.Type,
			&e.Payload,
			&e.CreatedAt,
		)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to scan event row")
			r.logger.ErrorContext(ctx, "failed to scan event row",
				"error", err,
				"limit", limit,
			)
			return nil, err
		}

		events = append(events, e)
	}

	if err := rows.Err(); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "rows iteration error")
		r.logger.ErrorContext(ctx, "rows iteration error",
			"error", err,
			"limit", limit,
		)
		return nil, err
	}

	span.SetAttributes(attribute.Int("event_count", len(events)))
	span.SetStatus(codes.Ok, "events retrieved successfully")
	return events, nil
}
