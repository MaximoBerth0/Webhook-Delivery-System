package postgres

import (
	"context"
	"log/slog"
	"webhook-delivery-system/internal/attempt"
	"webhook-delivery-system/internal/storage"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type AttemptRepository struct {
	db     storage.DBTX
	logger *slog.Logger
	tracer trace.Tracer
}

func NewAttemptRepository(db storage.DBTX, logger *slog.Logger) *AttemptRepository {
	return &AttemptRepository{
		db:     db,
		logger: logger,
		tracer: otel.Tracer("repository.attempt"),
	}
}

func (r *AttemptRepository) Create(ctx context.Context, a *attempt.DeliveryAttempt) error {
	ctx, span := r.tracer.Start(ctx, "AttemptRepository.Create")
	defer span.End()

	span.SetAttributes(
		attribute.String("delivery_id", a.DeliveryID),
		attribute.Int("attempt_number", a.AttemptNumber),
	)

	query := `
        INSERT INTO delivery_attempts (
            delivery_id, attempt_number, status, 
            response_code, error_message, created_at
        )
        VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING id
    `

	err := r.db.QueryRow(ctx, query,
		a.DeliveryID,
		a.AttemptNumber,
		a.Status,
		a.ResponseCode,
		a.ErrorMessage,
		a.CreatedAt,
	).Scan(&a.ID)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create delivery attempt")
		r.logger.ErrorContext(ctx, "failed to create attempt",
			"error", err,
			"delivery_id", a.DeliveryID,
		)
		return err
	}

	span.SetStatus(codes.Ok, "attempt created successfully")
	return nil
}

func (r *AttemptRepository) GetByDeliveryID(ctx context.Context, deliveryID string) ([]*attempt.DeliveryAttempt, error) {
	ctx, span := r.tracer.Start(ctx, "AttemptRepository.GetByDeliveryID")
	defer span.End()

	span.SetAttributes(attribute.String("delivery_id", deliveryID))

	query := `
        SELECT id, delivery_id, attempt_number, status, response_code, error_message, created_at
        FROM delivery_attempts
        WHERE delivery_id = $1
        ORDER BY attempt_number ASC
    `

	rows, err := r.db.Query(ctx, query, deliveryID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to query attempts")
		r.logger.ErrorContext(ctx, "failed to query delivery attempts",
			"error", err,
			"delivery_id", deliveryID,
		)
		return nil, err
	}
	defer rows.Close()

	var attempts []*attempt.DeliveryAttempt
	for rows.Next() {
		var a attempt.DeliveryAttempt
		err := rows.Scan(&a.ID, &a.DeliveryID, &a.AttemptNumber,
			&a.Status, &a.ResponseCode, &a.ErrorMessage, &a.CreatedAt)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to scan row")
			r.logger.ErrorContext(ctx, "failed to scan attempt row",
				"error", err,
				"delivery_id", deliveryID,
			)
			return nil, err
		}
		attempts = append(attempts, &a)
	}

	if err := rows.Err(); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "rows iteration error")
		r.logger.ErrorContext(ctx, "error iterating attempts rows",
			"error", err,
			"delivery_id", deliveryID,
		)
		return nil, err
	}

	span.SetAttributes(attribute.Int("attempts_count", len(attempts)))
	span.SetStatus(codes.Ok, "success")

	r.logger.DebugContext(ctx, "fetched delivery attempts",
		"delivery_id", deliveryID,
		"count", len(attempts),
	)

	return attempts, nil
}
