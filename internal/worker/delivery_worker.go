package worker

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
	"webhook-delivery-system/internal/attempt"
	"webhook-delivery-system/internal/delivery"
	"webhook-delivery-system/internal/event"
	"webhook-delivery-system/internal/webhook"
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
}

func NewDeliveryWorker(
	deliverySvc deliveryService,
	webhookRepo webhook.WebhookRepository,
	eventRepo event.EventRepository,
	attemptSvc *attempt.Service,
	signer Signer,
) *DeliveryWorker {
	return &DeliveryWorker{
		deliverySvc: deliverySvc,
		webhookRepo: webhookRepo,
		eventRepo:   eventRepo,
		attemptSvc:  attemptSvc,
		signer:      signer,
		httpClient:  &http.Client{Timeout: 10 * time.Second},
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
			log.Printf("delivery worker: error fetching pending: %v", err)
			w.sleep(ctx)
			continue
		}

		if len(deliveries) == 0 {
			w.sleep(ctx)
			continue
		}

		for _, d := range deliveries {
			if err := w.processDelivery(ctx, &d); err != nil {
				log.Printf("delivery worker: delivery %s failed: %v", d.ID, err)
			}
		}
	}
}

func (w *DeliveryWorker) processDelivery(ctx context.Context, d *delivery.Delivery) error {
	wh, err := w.webhookRepo.GetByID(ctx, d.WebhookID)
	if err != nil {
		return fmt.Errorf("get webhook %s: %w", d.WebhookID, err)
	}

	ev, err := w.eventRepo.GetByID(ctx, d.EventID)
	if err != nil {
		return fmt.Errorf("get event %s: %w", d.EventID, err)
	}

	maxAttempts := wh.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 1
	}

	var lastErr error
	for i := 0; i < maxAttempts; i++ {
		if err := w.attemptSvc.CreateAttempt(ctx, d.ID); err != nil {
			log.Printf("processDelivery: record attempt for %s: %v", d.ID, err)
		}
		if _, err := w.deliverySvc.IncrementAttempts(ctx, d.ID); err != nil {
			log.Printf("processDelivery: increment attempts for %s: %v", d.ID, err)
		}

		statusCode, err := w.dispatch(ctx, wh.TargetURL, wh.Secret, ev.Payload)
		if err != nil {
			lastErr = err
			log.Printf("processDelivery: attempt %d for %s: %v", i+1, d.ID, err)
			continue
		}

		if statusCode >= 200 && statusCode < 300 {
			if _, err := w.deliverySvc.MarkSuccess(ctx, d.ID); err != nil {
				return fmt.Errorf("mark delivery %s success: %w", d.ID, err)
			}
			return nil
		}

		lastErr = fmt.Errorf("non-2xx response: %d", statusCode)
		log.Printf("processDelivery: attempt %d for %s got status %d", i+1, d.ID, statusCode)
	}

	if _, err := w.deliverySvc.MarkFailed(ctx, d.ID); err != nil {
		return fmt.Errorf("mark delivery %s failed: %w", d.ID, err)
	}
	return lastErr
}

func (w *DeliveryWorker) dispatch(ctx context.Context, url, secret string, payload []byte) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return 0, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	if secret != "" {
		req.Header.Set("X-Webhook-Signature", w.signer.Sign(payload))
	}

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("http post: %w", err)
	}
	defer resp.Body.Close()

	return resp.StatusCode, nil
}

func (w *DeliveryWorker) sleep(ctx context.Context) {
	select {
	case <-ctx.Done():
	case <-time.After(workerPollInterval):
	}
}
