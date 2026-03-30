package postgres

import (
	"context"
	"webhook-delivery-system/internal/attempt"
	"webhook-delivery-system/internal/storage"
)

type AttemptRepository struct {
	db storage.DBTX
}

func NewAttemptRepository(db storage.DBTX) *AttemptRepository {
	return &AttemptRepository{db: db}
}

func (r *AttemptRepository) Create(ctx context.Context, a *attempt.DeliveryAttempt) error {
	query := `
		INSERT INTO delivery_attempts (id, delivery_id, attempt_number, status, response_code, error_message, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.Exec(ctx, query,
		a.ID,
		a.DeliveryID,
		a.AttemptNumber,
		a.Status,
		a.ResponseCode,
		a.ErrorMessage,
		a.CreatedAt,
	)
	return err
}

func (r *AttemptRepository) GetByDeliveryID(ctx context.Context, deliveryID string) ([]attempt.DeliveryAttempt, error) {
	query := `
		SELECT id, delivery_id, attempt_number, status, response_code, error_message, created_at
		FROM delivery_attempts
		WHERE delivery_id = $1
		ORDER BY attempt_number ASC
	`
	rows, err := r.db.Query(ctx, query, deliveryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var attempts []attempt.DeliveryAttempt
	for rows.Next() {
		var a attempt.DeliveryAttempt
		err := rows.Scan(&a.ID, &a.DeliveryID, &a.AttemptNumber, &a.Status, &a.ResponseCode, &a.ErrorMessage, &a.CreatedAt)
		if err != nil {
			return nil, err
		}
		attempts = append(attempts, a)
	}
	return attempts, rows.Err()
}
