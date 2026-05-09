package worker

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"
	"webhook-delivery-system/internal/attempt"
	"webhook-delivery-system/internal/delivery"
	"webhook-delivery-system/internal/event"
	"webhook-delivery-system/internal/webhook"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type Signer interface {
	Sign(payload []byte) string
}

type deliveryService interface {
	GetPending(ctx context.Context, limit int) ([]delivery.Delivery, error)
	GetByID(ctx context.Context, id string) (*delivery.Delivery, error)
	MarkSuccess(ctx context.Context, id string) (*delivery.Delivery, error)
	MarkFailed(ctx context.Context, id string) (*delivery.Delivery, error)
}

type attemptService interface {
	CreateAttempt(ctx context.Context, deliveryID string) (*attempt.DeliveryAttempt, error)
	CreateScheduledAttempt(ctx context.Context, deliveryID string, scheduledFor time.Time) error
	GetReadyAttempts(ctx context.Context) ([]*attempt.DeliveryAttempt, error)
	GetAttempts(ctx context.Context, deliveryID string) ([]attempt.DeliveryAttempt, error)
	UpdateAttemptStatus(ctx context.Context, attempt *attempt.DeliveryAttempt) error
	CalculateNextAttemptTime(attemptNumber int, failedAt time.Time) time.Time
}

const (
	workerPollInterval = 5 * time.Second
	workerBatchSize    = 5
)

type DeliveryWorker struct {
	deliverySvc deliveryService
	webhookRepo webhook.WebhookRepository
	eventRepo   event.EventRepository
	attemptSvc  attemptService
	signer      Signer
	httpClient  *http.Client
	logger      *slog.Logger
	tracer      trace.Tracer
}

func NewDeliveryWorker(
	deliverySvc deliveryService,
	webhookRepo webhook.WebhookRepository,
	eventRepo event.EventRepository,
	attemptSvc attemptService,
	signer Signer,
	logger *slog.Logger,
) *DeliveryWorker {
	return &DeliveryWorker{
		deliverySvc: deliverySvc,
		webhookRepo: webhookRepo,
		eventRepo:   eventRepo,
		attemptSvc:  attemptSvc,
		signer:      signer,
		httpClient:  &http.Client{Timeout: 10 * time.Second},
		logger:      logger,
		tracer:      otel.Tracer("worker.delivery"),
	}
}

func (w *DeliveryWorker) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if err := w.ProcessScheduledAttempts(ctx); err != nil {
			w.logger.ErrorContext(ctx, "error processing scheduled attempts", "error", err)
		}

		deliveries, err := w.deliverySvc.GetPending(ctx, workerBatchSize)
		if err != nil {
			w.logger.ErrorContext(ctx, "error fetching pending deliveries", "error", err)
			w.sleep(ctx)
			continue
		}

		if len(deliveries) == 0 {
			w.sleep(ctx)
			continue
		}

		for _, d := range deliveries {
			if err := w.ProcessFirstAttempt(ctx, &d); err != nil {
				w.logger.ErrorContext(ctx, "first attempt failed", "delivery_id", d.ID, "error", err)
			}
		}
	}
}

func (w *DeliveryWorker) ProcessScheduledAttempts(ctx context.Context) error {
	attempts, err := w.attemptSvc.GetReadyAttempts(ctx)
	if err != nil {
		return err
	}

	for _, att := range attempts {
		if err := w.processAttempt(ctx, att); err != nil {
			w.logger.ErrorContext(ctx, "scheduled attempt failed",
				"attempt_id", att.ID,
				"delivery_id", att.DeliveryID,
				"error", err,
			)
		}
	}

	return nil
}

func (w *DeliveryWorker) ProcessFirstAttempt(ctx context.Context, d *delivery.Delivery) error {
	ctx, span := w.tracer.Start(ctx, "DeliveryWorker.processFirstAttempt")
	defer span.End()

	span.SetAttributes(
		attribute.String("delivery_id", d.ID),
		attribute.String("webhook_id", d.WebhookID),
		attribute.String("event_id", d.EventID),
	)

	currentAttempt, err := w.attemptSvc.CreateAttempt(ctx, d.ID)
	if err != nil {
		return err
	}

	return w.processAttempt(ctx, currentAttempt)
}

func (w *DeliveryWorker) processAttempt(ctx context.Context, att *attempt.DeliveryAttempt) error {
	ctx, span := w.tracer.Start(ctx, "DeliveryWorker.processAttempt")
	defer span.End()

	span.SetAttributes(
		attribute.String("attempt_id", att.ID),
		attribute.String("delivery_id", att.DeliveryID),
		attribute.Int("attempt_number", att.AttemptNumber),
	)

	del, err := w.deliverySvc.GetByID(ctx, att.DeliveryID)
	if err != nil {
		w.logger.ErrorContext(ctx, "failed to get delivery",
			"delivery_id", att.DeliveryID,
			"error", err,
		)
		return err
	}

	wh, err := w.webhookRepo.GetByID(ctx, del.WebhookID)
	if err != nil {
		w.logger.ErrorContext(ctx, "failed to get webhook",
			"webhook_id", del.WebhookID,
			"error", err,
		)
		return err
	}

	ev, err := w.eventRepo.GetByID(ctx, del.EventID)
	if err != nil {
		w.logger.ErrorContext(ctx, "failed to get event",
			"event_id", del.EventID,
			"error", err,
		)
		return err
	}

	maxAttempts := wh.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 5
	}

	statusCode, err := w.dispatch(ctx, wh.TargetURL, wh.Secret, ev.Payload)

	if err == nil && statusCode >= 200 && statusCode < 300 {
		if err := att.MarkAsSuccess(statusCode); err != nil {
			return err
		}

		if err := w.attemptSvc.UpdateAttemptStatus(ctx, att); err != nil {
			return err
		}

		if _, err := w.deliverySvc.MarkSuccess(ctx, att.DeliveryID); err != nil {
			return err
		}

		return nil
	}

	errorMsg := ""
	if err != nil {
		errorMsg = err.Error()
	} else {
		errorMsg = "non-2xx response"
	}

	if err := att.MarkAsFailed(statusCode, errorMsg); err != nil {
		return err
	}

	if err := w.attemptSvc.UpdateAttemptStatus(ctx, att); err != nil {
		return err
	}

	if att.AttemptNumber < maxAttempts {
		nextScheduledTime := w.attemptSvc.CalculateNextAttemptTime(
			att.AttemptNumber,
			time.Now(),
		)
		return w.attemptSvc.CreateScheduledAttempt(ctx, att.DeliveryID, nextScheduledTime)
	}

	if _, err := w.deliverySvc.MarkFailed(ctx, att.DeliveryID); err != nil {
		return err
	}

	return errors.New("max attempts reached")
}

func (w *DeliveryWorker) dispatch(ctx context.Context, url, secret string, payload []byte) (int, error) {
	ctx, span := w.tracer.Start(ctx, "DeliveryWorker.dispatch")
	defer span.End()

	span.SetAttributes(attribute.String("target_url", url))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create request")
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")

	if secret != "" {
		req.Header.Set("X-Webhook-Signature", w.signer.Sign(payload))
	}

	resp, err := w.httpClient.Do(req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "http post failed")
		return 0, err
	}
	defer resp.Body.Close()

	span.SetAttributes(attribute.Int("status_code", resp.StatusCode))
	span.SetStatus(codes.Ok, "dispatch completed")
	return resp.StatusCode, nil
}

func (w *DeliveryWorker) sleep(ctx context.Context) {
	select {
	case <-ctx.Done():
	case <-time.After(workerPollInterval):
	}
}
