package postgres

import (
	"context"
	"errors"
	"log/slog"
	"webhook-delivery-system/internal/delivery"
	"webhook-delivery-system/internal/storage"

	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type DeliveryRepository struct {
	db     storage.DBTX
	logger *slog.Logger
	tracer trace.Tracer
}

func NewDeliveryRepository(db storage.DBTX, logger *slog.Logger) *DeliveryRepository {
	return &DeliveryRepository{db: db, logger: logger, tracer: otel.Tracer("repository.delivery")}
}

func (r *DeliveryRepository) Create(ctx context.Context, d *delivery.Delivery) error {
	ctx, span := r.tracer.Start(ctx, "DeliveryRepository.Create")
	defer span.End()

	query := `
        INSERT INTO deliveries (event_id, webhook_id, attempts, status)
        VALUES ($1, $2, $3, $4)
        RETURNING id
    `
	err := r.db.QueryRow(
		ctx,
		query,
		d.EventID,
		d.WebhookID,
		d.Attempts,
		d.Status,
	).Scan(&d.ID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create delivery")
		r.logger.ErrorContext(ctx, "failed to create delivery",
			"error", err,
			"event_id", d.EventID,
			"webhook_id", d.WebhookID,
		)
		return err
	}

	span.SetAttributes(attribute.String("delivery_id", d.ID))
	span.SetStatus(codes.Ok, "delivery created successfully")
	return nil
}

func (r *DeliveryRepository) GetByID(ctx context.Context, id string) (*delivery.Delivery, error) {
	ctx, span := r.tracer.Start(ctx, "DeliveryRepository.GetByID")
	defer span.End()

	span.SetAttributes(attribute.String("delivery_id", id))

	query := `SELECT id, event_id, webhook_id, attempts, status FROM deliveries WHERE id = $1`

	d := &delivery.Delivery{}

	err := r.db.QueryRow(ctx, query, id).
		Scan(&d.ID, &d.EventID, &d.WebhookID, &d.Attempts, &d.Status)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to query delivery")
		r.logger.ErrorContext(ctx, "failed to query id",
			"error", err,
			"delivery_id", id,
		)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, storage.ErrDeliveryNotFound
		}
		return nil, err
	}
	span.SetAttributes(attribute.String("delivery_status", string(d.Status)))

	return d, nil
}

func (r *DeliveryRepository) GetByWebhookID(ctx context.Context, webhookID string) ([]delivery.Delivery, error) {
	ctx, span := r.tracer.Start(ctx, "DeliveryRepository.GetByWebhookID")
	defer span.End()

	query := `SELECT id, event_id, webhook_id, attempts, status FROM deliveries WHERE webhook_id = $1`

	span.SetAttributes(attribute.String("webhook_id", webhookID))

	rows, err := r.db.Query(ctx, query, webhookID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to query deliveries")
		r.logger.ErrorContext(ctx, "failed to query deliveries by webhook",
			"error", err,
			"webhook_id", webhookID,
		)
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
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to scan delivery row")
			r.logger.ErrorContext(ctx, "failed to scan delivery row",
				"error", err,
				"webhook_id", webhookID,
			)
			return nil, err
		}
		deliveries = append(deliveries, d)
	}

	if err := rows.Err(); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "rows iteration error")
		r.logger.ErrorContext(ctx, "rows iteration error",
			"error", err,
			"webhook_id", webhookID,
		)
		return nil, err
	}

	span.SetAttributes(attribute.Int("delivery_count", len(deliveries)))
	span.SetStatus(codes.Ok, "deliveries retrieved successfully")
	return deliveries, nil
}

func (r *DeliveryRepository) GetPending(ctx context.Context, limit int) ([]delivery.Delivery, error) {
	// locked by a concurrent worker (FOR UPDATE SKIP LOCKED)

	ctx, span := r.tracer.Start(ctx, "DeliveryRepository.GetPending")
	defer span.End()

	span.SetAttributes(
		attribute.Int("limit", limit),
	)

	query := `
        UPDATE deliveries
        SET status = 'PROCESSING'
        WHERE id IN (
            SELECT id
            FROM deliveries
            WHERE status = 'PENDING'
            ORDER BY created_at ASC
            LIMIT $1
            FOR UPDATE SKIP LOCKED
        )
        RETURNING id, event_id, webhook_id, attempts, status
    `

	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to query pending deliveries")
		r.logger.ErrorContext(ctx, "failed to query pending deliveries",
			"error", err,
			"limit", limit,
		)
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
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to scan pending deliveries")
			r.logger.ErrorContext(ctx, "failed to scan pending deliveries",
				"error", err,
				"limit", limit,
			)
			return nil, err
		}
		deliveries = append(deliveries, d)
	}

	if err := rows.Err(); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "row iteration error")
		r.logger.ErrorContext(ctx, "row iteration error",
			"error", err,
			"limit", limit,
		)
		return nil, err
	}

	return deliveries, nil
}

func (r *DeliveryRepository) UpdateStatus(ctx context.Context, id string, status delivery.Status) (*delivery.Delivery, error) {
	ctx, span := r.tracer.Start(ctx, "DeliveryRepository.UpdateStatus")
	defer span.End()

	span.SetAttributes(
		attribute.String("delivery_id", id),
		attribute.String("new_status", string(status)),
	)

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
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to update delivery status")
		r.logger.ErrorContext(ctx, "failed to update delivery status",
			"error", err,
			"delivery_id", id,
			"new_status", string(status),
		)

		if errors.Is(err, pgx.ErrNoRows) {
			return nil, storage.ErrDeliveryNotFound
		}

		return nil, err
	}
	return &d, nil
}
