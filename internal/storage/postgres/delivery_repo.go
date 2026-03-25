package postgres

import (
	"context"
	"errors"
	"webhook-delivery-system/internal/delivery"
	"webhook-delivery-system/internal/storage"

	"github.com/jackc/pgx/v5"
)

type DeliveryRepository struct {
	db storage.DBTX
}

func NewDeliveryRepository(db storage.DBTX) *DeliveryRepository {
	return &DeliveryRepository{db: db}
}

func (r *DeliveryRepository) Create(ctx context.Context, d *delivery.Delivery) error {
	query := `
		INSERT INTO deliveries (id, event_id, webhook_id, attempts, status)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := r.db.Exec(
		ctx,
		query,
		d.ID,
		d.EventID,
		d.WebhookID,
		d.Attempts,
		d.Status,
	)

	return err
}

func (r *DeliveryRepository) GetByID(ctx context.Context, id string) (*delivery.Delivery, error) {
	query := `SELECT id, event_id, webhook_id, attempts, status FROM deliveries WHERE id = $1`

	d := &delivery.Delivery{}

	err := r.db.QueryRow(ctx, query, id).
		Scan(&d.ID, &d.EventID, &d.WebhookID, &d.Attempts, &d.Status)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, storage.ErrDeliveryNotFound
		}
		return nil, err
	}

	return d, nil
}

func (r *DeliveryRepository) GetPending(ctx context.Context, limit int) ([]delivery.Delivery, error) {
	query := `
		SELECT id, event_id, webhook_id, attempts, status
		FROM deliveries
		WHERE status in ('PENDING')
		ORDER BY created_at ASC
		LIMIT $1
	`

	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deliveries []delivery.Delivery

	for rows.Next() {
		var d delivery.Delivery

		err := rows.Scan(
			&d.ID,
			&d.EventID,
			&d.WebhookID,
			&d.Attempts,
			&d.Status,
		)
		if err != nil {
			return nil, err
		}

		deliveries = append(deliveries, d)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return deliveries, nil
}

func (r *DeliveryRepository) GetRetryable(ctx context.Context, limit int) ([]delivery.Delivery, error) {
	query := `
	    SELECT id, event_id, webhook_id, attempts, status
		FROM deliveries
		WHERE status IN ('PENDING', 'RETRY')
        AND attempts < $2
		ORDER BY created_at ASC
		LIMIT $1
	`
	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deliveries []delivery.Delivery

	for rows.Next() {
		var d delivery.Delivery

		err := rows.Scan(
			&d.ID,
			&d.EventID,
			&d.WebhookID,
			&d.Attempts,
			&d.Status,
		)
		if err != nil {
			return nil, err
		}

		deliveries = append(deliveries, d)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return deliveries, nil
}

func (r *DeliveryRepository) UpdateStatus(ctx context.Context, id string, status delivery.Status) (*delivery.Delivery, error) {
	query := `
	    UPDATE deliveries
	    SET status = $1
	    WHERE id = $2
	    RETURNING id, event_id, webhook_id, attempts, status
        `

	var d delivery.Delivery

	err := r.db.QueryRow(ctx, query, status, id).Scan(
		&d.ID,
		&d.EventID,
		&d.WebhookID,
		&d.Attempts,
		&d.Status,
	)
	if err != nil {
		return nil, err
	}

	return &d, nil
}

func (r *DeliveryRepository) IncrementAttempts(ctx context.Context, id string) (*delivery.Delivery, error) {
	query := `
	    UPDATE deliveries
        SET attempts = attempts + 1
        WHERE id = $1
        RETURNING id, event_id, webhook_id, attempts, status
		`

	var d delivery.Delivery

	err := r.db.QueryRow(ctx, query, id).Scan(
		&d.ID,
		&d.EventID,
		&d.WebhookID,
		&d.Attempts,
		&d.Status,
	)
	if err != nil {
		return nil, err
	}

	return &d, nil
}
