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
	MarkSuccess(ctx context.Context, id string) (*delivery.Delivery, error)
	MarkFailed(ctx context.Context, id string) (*delivery.Delivery, error)
	IncrementAttempts(ctx context.Context, id string) (*delivery.Delivery, error)
}

const (
	workerPollInterval = 5 * time.Second
	workerBatchSize    = 10
)

type DeliveryWorker struct {
	deliverySvc deliveryService
	webhookRepo webhook.WebhookRepository
	eventRepo   event.EventRepository
	attemptSvc  *attempt.Service
	signer      Signer
	httpClient  *http.Client
	logger      *slog.Logger
	tracer      trace.Tracer
}

func NewDeliveryWorker(
	deliverySvc deliveryService,
	webhookRepo webhook.WebhookRepository,
	eventRepo event.EventRepository,
	attemptSvc *attempt.Service,
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
			if err := w.processDelivery(ctx, &d); err != nil {
				w.logger.ErrorContext(ctx, "delivery failed", "delivery_id", d.ID, "error", err)
			}
		}
	}
}

func (w *DeliveryWorker) processDelivery(ctx context.Context, d *delivery.Delivery) error {
	ctx, span := w.tracer.Start(ctx, "DeliveryWorker.processDelivery")
	defer span.End()

	span.SetAttributes(
		attribute.String("delivery_id", d.ID),
		attribute.String("webhook_id", d.WebhookID),
		attribute.String("event_id", d.EventID),
	)

	wh, err := w.webhookRepo.GetByID(ctx, d.WebhookID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get webhook")
		w.logger.ErrorContext(ctx, "failed to get webhook", "webhook_id", d.WebhookID, "error", err)
		return err
	}

	ev, err := w.eventRepo.GetByID(ctx, d.EventID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get event")
		w.logger.ErrorContext(ctx, "failed to get event", "event_id", d.EventID, "error", err)
		return err
	}

	maxAttempts := wh.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 1
	}

	var lastErr error
	for i := 0; i < maxAttempts; i++ {
		if err := w.attemptSvc.CreateAttempt(ctx, d.ID); err != nil {
			w.logger.ErrorContext(ctx, "failed to record attempt",
				"delivery_id", d.ID,
				"attempt", i+1,
				"error", err,
			)
		}
		if _, err := w.deliverySvc.IncrementAttempts(ctx, d.ID); err != nil {
			w.logger.ErrorContext(ctx, "failed to increment attempts",
				"delivery_id", d.ID,
				"attempt", i+1,
				"error", err,
			)
		}

		statusCode, err := w.dispatch(ctx, wh.TargetURL, wh.Secret, ev.Payload)
		if err != nil {
			lastErr = err
			w.logger.ErrorContext(ctx, "dispatch failed",
				"delivery_id", d.ID,
				"attempt", i+1,
				"error", err,
			)
			continue
		}

		if statusCode >= 200 && statusCode < 300 {
			if _, err := w.deliverySvc.MarkSuccess(ctx, d.ID); err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, "failed to mark delivery success")
				w.logger.ErrorContext(ctx, "failed to mark delivery success", "delivery_id", d.ID, "error", err)
				return err
			}
			span.SetStatus(codes.Ok, "delivery successful")
			return nil
		}

		lastErr = errors.New("non-2xx response")
		w.logger.ErrorContext(ctx, "non-2xx response",
			"delivery_id", d.ID,
			"attempt", i+1,
			"status_code", statusCode,
		)
	}

	if _, err := w.deliverySvc.MarkFailed(ctx, d.ID); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to mark delivery failed")
		w.logger.ErrorContext(ctx, "failed to mark delivery failed", "delivery_id", d.ID, "error", err)
		return err
	}

	span.SetStatus(codes.Error, "delivery exhausted all attempts")
	return lastErr
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
