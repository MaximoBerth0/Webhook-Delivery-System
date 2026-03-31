package postgres

import (
	"context"
	"webhook-delivery-system/internal/event"
	"webhook-delivery-system/internal/storage"
	"webhook-delivery-system/internal/webhook"
)

type WebhookRepository struct {
	db storage.DBTX
}

func NewWebhookRepository(db storage.DBTX) *WebhookRepository {
	return &WebhookRepository{db: db}
}

func (r *WebhookRepository) Create(ctx context.Context, w *webhook.Webhook) error {
	query := `
		INSERT INTO webhook (id, target_url, subscribed_events, secret, active, max_attempts, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.db.Exec(ctx, query,
		w.ID,
		w.TargetURL,
		w.SubscribedEvents,
		w.Secret,
		w.Active,
		w.MaxAttempts,
		w.CreatedAt,
		w.UpdatedAt,
	)
	return err
}

func (r *WebhookRepository) GetByID(ctx context.Context, id string) (*webhook.Webhook, error) {
	query := `
		SELECT id, target_url, subscribed_events, secret, active, max_attempts, created_at, updated_at
		FROM webhook
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
		return nil, err
	}
	return &w, nil
}

func (r *WebhookRepository) ListByEvent(ctx context.Context, event event.SubscribedEvent) ([]webhook.Webhook, error) {
	query := `
        SELECT id, target_url, subscribed_events, secret, active, max_attempts, created_at, updated_at
        FROM webhook
        WHERE $1 = ANY(subscribed_events)
        AND active = true
    `
	rows, err := r.db.Query(ctx, query, event)
	if err != nil {
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
			return nil, err
		}
		webhooks = append(webhooks, w)
	}

	return webhooks, nil
}

func (r *WebhookRepository) Update(ctx context.Context, w *webhook.Webhook) error {
	query := `
		UPDATE webhook
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
	return err
}

func (r *WebhookRepository) Delete(ctx context.Context, id string) error {
	query := `
        UPDATE webhook
        SET active = false, updated_at = NOW()
        WHERE id = $1
    `
	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrWebhookNotFound
	}

	return nil
}
