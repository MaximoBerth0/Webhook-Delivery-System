package postgres

import (
	"context"
	"log/slog"
	"time"
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

func (r *AttemptRepository) CreateAttempt(ctx context.Context, a *attempt.DeliveryAttempt) error {
	ctx, span := r.tracer.Start(ctx, "AttemptRepository.Create")
	defer span.End()

	span.SetAttributes(
		attribute.String("delivery_id", a.DeliveryID),
		attribute.Int("attempt_number", a.AttemptNumber),
	)

	query := `
        INSERT INTO delivery_attempts (
            id, delivery_id, attempt_number, status, scheduled_for,
            response_code, error_message, created_at
        )
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
    `

	_, err := r.db.Exec(ctx, query,
		a.ID,
		a.DeliveryID,
		a.AttemptNumber,
		a.Status,
		a.ScheduledFor,
		a.ResponseCode,
		a.ErrorMessage,
		a.CreatedAt,
	)

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

func (r *AttemptRepository) GetAttempts(ctx context.Context, deliveryID string) ([]attempt.DeliveryAttempt, error) {
	ctx, span := r.tracer.Start(ctx, "AttemptRepository.GetAttempts")
	defer span.End()

	span.SetAttributes(attribute.String("delivery_id", deliveryID))

	query := `
        SELECT id, delivery_id, attempt_number, status, scheduled_for,
               response_code, error_message, created_at, executed_at
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

	var attempts []attempt.DeliveryAttempt
	for rows.Next() {
		var a attempt.DeliveryAttempt
		err := rows.Scan(
			&a.ID, &a.DeliveryID, &a.AttemptNumber,
			&a.Status, &a.ScheduledFor, &a.ResponseCode,
			&a.ErrorMessage, &a.CreatedAt, &a.ExecutedAt,
		)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to scan row")
			r.logger.ErrorContext(ctx, "failed to scan attempt row",
				"error", err,
				"delivery_id", deliveryID,
			)
			return nil, err
		}
		attempts = append(attempts, a)
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

func (r *AttemptRepository) GetReadyAttempts(ctx context.Context, before time.Time) ([]*attempt.DeliveryAttempt, error) {
	ctx, span := r.tracer.Start(ctx, "AttemptRepository.GetReadyAttempts")
	defer span.End()

	query := `
        SELECT id, delivery_id, attempt_number, status, scheduled_for,
               response_code, error_message, created_at, executed_at
        FROM delivery_attempts
        WHERE status = $1 AND scheduled_for <= $2
        ORDER BY scheduled_for ASC
    `

	rows, err := r.db.Query(ctx, query, attempt.StatusPending, before)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to query ready attempts")
		r.logger.ErrorContext(ctx, "failed to query ready attempts", "error", err)
		return nil, err
	}
	defer rows.Close()

	var attempts []*attempt.DeliveryAttempt
	for rows.Next() {
		var a attempt.DeliveryAttempt
		err := rows.Scan(
			&a.ID, &a.DeliveryID, &a.AttemptNumber,
			&a.Status, &a.ScheduledFor, &a.ResponseCode,
			&a.ErrorMessage, &a.CreatedAt, &a.ExecutedAt,
		)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to scan row")
			r.logger.ErrorContext(ctx, "failed to scan ready attempt row", "error", err)
			return nil, err
		}
		attempts = append(attempts, &a)
	}

	if err := rows.Err(); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "rows iteration error")
		r.logger.ErrorContext(ctx, "error iterating ready attempts rows", "error", err)
		return nil, err
	}

	span.SetAttributes(attribute.Int("ready_attempts_count", len(attempts)))
	span.SetStatus(codes.Ok, "success")
	return attempts, nil
}

func (r *AttemptRepository) UpdateStatus(ctx context.Context, a *attempt.DeliveryAttempt) error {
	ctx, span := r.tracer.Start(ctx, "AttemptRepository.UpdateStatus")
	defer span.End()

	span.SetAttributes(
		attribute.String("attempt_id", a.ID),
		attribute.String("status", string(a.Status)),
	)

	query := `
        UPDATE delivery_attempts
        SET status = $1, response_code = $2, error_message = $3, executed_at = $4
        WHERE id = $5
    `

	_, err := r.db.Exec(ctx, query, a.Status, a.ResponseCode, a.ErrorMessage, a.ExecutedAt, a.ID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to update attempt status")
		r.logger.ErrorContext(ctx, "failed to update attempt status",
			"error", err,
			"attempt_id", a.ID,
		)
		return err
	}

	span.SetStatus(codes.Ok, "attempt status updated successfully")
	return nil
}
