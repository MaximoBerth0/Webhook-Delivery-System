package http

import (
	"net/http"

	"github.com/go-chi/chi"
)

func NewRouter(delivery *DeliveryHandler, event *EventHandler, webhook *WebhookHandler) http.Handler {
	r := chi.NewRouter()

	r.Post("/events", event.CreateEvent)
	r.Get("/events", event.ListEvents)

	r.Post("/webhooks", webhook.CreateWebhook)
	r.Get("/webhooks/{id}", webhook.GetByID)
	r.Put("/webhooks/{id}", webhook.UpdateWebhook)
	r.Delete("/webhooks/{id}", webhook.Delete)

	r.Get("/deliveries/{id}", delivery.GetByID)
	r.Get("/webhooks/{id}/deliveries", delivery.ListByWebhook)
	r.Get("/deliveries/{id}/attempts", delivery.GetAttempts)

	return r
}
