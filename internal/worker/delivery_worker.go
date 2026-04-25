package worker

import (
	"net/http"
	"time"

	"webhook-delivery-system/internal/attempt"
	"webhook-delivery-system/internal/delivery"
	"webhook-delivery-system/internal/event"
	"webhook-delivery-system/internal/webhook"
)

type DeliveryWorker struct {
	deliveryRepo delivery.DeliveryRepository
	webhookRepo  webhook.WebhookRepository
	eventRepo    event.EventRepository
	attemptSvc   attempt.Service
	httpClient   *http.Client
}

func NewDeliveryWorker(
	deliveryRepo delivery.DeliveryRepository,
	webhookRepo webhook.WebhookRepository,
	eventRepo event.EventRepository,
	attemptSvc attempt.Service,
) *DeliveryWorker {
	return &DeliveryWorker{
		deliveryRepo: deliveryRepo,
		webhookRepo:  webhookRepo,
		eventRepo:    eventRepo,
		attemptSvc:   attemptSvc,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
	}
}
