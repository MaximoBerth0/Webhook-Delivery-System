package postgres

import (
	"context"
	"errors"
	"log/slog"
	"webhook-delivery-system/internal/event"
	"webhook-delivery-system/internal/storage"
	"webhook-delivery-system/internal/webhook"

	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type WebhookRepository struct {
	db     storage.DBTX
	logger *slog.Logger
	tracer trace.Tracer
}

func NewWebhookRepository(db storage.DBTX, logger *slog.Logger) *WebhookRepository {
	return &WebhookRepository{db: db, logger: logger, tracer: otel.Tracer("repository.webhook")}
}

func (r *WebhookRepository) Create(ctx context.Context, w *webhook.Webhook) error {
	ctx, span := r.tracer.Start(ctx, "WebhookRepository.Create")
	defer span.End()

	query := `
    INSERT INTO webhooks (target_url, subscribed_events, secret, active, max_attempts, created_at, updated_at)
    VALUES ($1, $2, $3, $4, $5, $6, $7)
    RETURNING id
	`
	err := r.db.QueryRow(ctx, query,
		w.TargetURL,
		w.SubscribedEvents,
		w.Secret,
		w.Active,
		w.MaxAttempts,
		w.CreatedAt,
		w.UpdatedAt,
	).Scan(&w.ID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create webhook")
		r.logger.ErrorContext(ctx, "failed to create webhook",
			"error", err,
			"target_url", w.TargetURL,
		)
		return err
	}

	span.SetAttributes(attribute.String("webhook_id", w.ID))
	span.SetStatus(codes.Ok, "webhook created successfully")
	return nil
}

func (r *WebhookRepository) GetByID(ctx context.Context, id string) (*webhook.Webhook, error) {
	ctx, span := r.tracer.Start(ctx, "WebhookRepository.GetByID")
	defer span.End()

	span.SetAttributes(attribute.String("webhook_id", id))

	query := `
		SELECT id, target_url, subscribed_events, secret, active, max_attempts, created_at, updated_at
		FROM webhooks
		WHERE id = $1
	`

	var w webhook.Webhook
	err := r.db.QueryRow(ctx, query, id).Scan(
		&w.ID,
		&w.TargetURL,
		&w.SubscribedEvents,
		&w.Secret,
		&w.Active,
		&w.MaxAttempts,
		&w.CreatedAt,
		&w.UpdatedAt,
	)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to query webhook")
		r.logger.ErrorContext(ctx, "failed to query webhook by id",
			"error", err,
			"webhook_id", id,
		)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, storage.ErrWebhookNotFound
		}
		return nil, err
	}

	span.SetStatus(codes.Ok, "webhook retrieved successfully")
	return &w, nil
}

func (r *WebhookRepository) ListByEvent(ctx context.Context, e event.SubscribedEvent) ([]webhook.Webhook, error) {
	ctx, span := r.tracer.Start(ctx, "WebhookRepository.ListByEvent")
	defer span.End()

	span.SetAttributes(attribute.String("event_type", string(e)))

	query := `
        SELECT id, target_url, subscribed_events, secret, active, max_attempts, created_at, updated_at
        FROM webhooks
        WHERE $1 = ANY(subscribed_events)
        AND active = true
    `
	rows, err := r.db.Query(ctx, query, e)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to query webhooks by event")
		r.logger.ErrorContext(ctx, "failed to query webhooks by event",
			"error", err,
			"event_type", string(e),
		)
		return nil, err
	}
	defer rows.Close()

	var webhooks []webhook.Webhook
	for rows.Next() {
		var w webhook.Webhook
		err := rows.Scan(
			&w.ID,
			&w.TargetURL,
			&w.SubscribedEvents,
			&w.Secret,
			&w.Active,
			&w.MaxAttempts,
			&w.CreatedAt,
			&w.UpdatedAt,
		)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to scan webhook row")
			r.logger.ErrorContext(ctx, "failed to scan webhook row",
				"error", err,
				"event_type", string(e),
			)
			return nil, err
		}
		webhooks = append(webhooks, w)
	}

	if err := rows.Err(); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "rows iteration error")
		r.logger.ErrorContext(ctx, "rows iteration error",
			"error", err,
			"event_type", string(e),
		)
		return nil, err
	}

	span.SetAttributes(attribute.Int("webhook_count", len(webhooks)))
	span.SetStatus(codes.Ok, "webhooks retrieved successfully")
	return webhooks, nil
}

func (r *WebhookRepository) Update(ctx context.Context, w *webhook.Webhook) error {
	ctx, span := r.tracer.Start(ctx, "WebhookRepository.Update")
	defer span.End()

	span.SetAttributes(attribute.String("webhook_id", w.ID))

	query := `
		UPDATE webhooks
		SET target_url = $1, subscribed_events = $2, secret = $3, active = $4, max_attempts = $5, updated_at = $6
		WHERE id = $7
	`

	_, err := r.db.Exec(ctx, query,
		w.TargetURL,
		w.SubscribedEvents,
		w.Secret,
		w.Active,
		w.MaxAttempts,
		w.UpdatedAt,
		w.ID,
	)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to update webhook")
		r.logger.ErrorContext(ctx, "failed to update webhook",
			"error", err,
			"webhook_id", w.ID,
		)
		return err
	}

	span.SetStatus(codes.Ok, "webhook updated successfully")
	return nil
}

func (r *WebhookRepository) Delete(ctx context.Context, id string) error {
	ctx, span := r.tracer.Start(ctx, "WebhookRepository.Delete")
	defer span.End()

	span.SetAttributes(attribute.String("webhook_id", id))

	query := `
        UPDATE webhooks
        SET active = false, updated_at = NOW()
        WHERE id = $1
    `
	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to delete webhook")
		r.logger.ErrorContext(ctx, "failed to delete webhook",
			"error", err,
			"webhook_id", id,
		)
		return err
	}

	if result.RowsAffected() == 0 {
		span.SetStatus(codes.Error, "webhook not found")
		return storage.ErrWebhookNotFound
	}

	span.SetStatus(codes.Ok, "webhook deleted successfully")
	return nil
}
