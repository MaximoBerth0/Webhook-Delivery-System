package postgres

import (
	"context"
	"time"

	"webhook-delivery-system/internal/event"
	"webhook-delivery-system/internal/storage"
)

type EventRepository struct {
	db storage.DBTX
}

func NewEventRepository(db storage.DBTX) *EventRepository {
	return &EventRepository{db: db}
}

func (r *EventRepository) Create(ctx context.Context, e *event.Event) error {
	query := `
		INSERT INTO events (id, type, payload, created_at)
		VALUES ($1, $2, $3, $4)
	`
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now()
	}

	_, err := r.db.Exec(
		ctx,
		query,
		e.ID,
		e.Type,
		e.Payload,
		e.CreatedAt,
	)

	return err
}

func (r *EventRepository) GetByID(ctx context.Context, id string) (*event.Event, error)

func (r *EventRepository) List(ctx context.Context, limit int) ([]event.Event, error)
