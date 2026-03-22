package postgres

import (
	"context"
	"errors"
	"time"
	"webhook-delivery-system/internal/event"
	"webhook-delivery-system/internal/storage"

	"github.com/jackc/pgx/v5"
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

func (r *EventRepository) GetByID(ctx context.Context, id string) (*event.Event, error) {
	query := `SELECT id, type, payload, created_at FROM events WHERE id = $1`

	e := new(event.Event)

	err := r.db.QueryRow(ctx, query, id).
		Scan(&e.ID, &e.Type, &e.Payload, &e.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, storage.ErrEventNotFound
		}
		return nil, err
	}

	return e, nil
}

func (r *EventRepository) List(ctx context.Context, limit int) ([]event.Event, error) {
	query := `
		SELECT id, type, payload, created_at
		FROM events
		ORDER BY created_at DESC
		LIMIT $1
	`

	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
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
			return nil, err
		}

		events = append(events, e)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return events, nil
}
