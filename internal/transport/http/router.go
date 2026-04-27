package http

import (
	"net/http"

	"webhook-delivery-system/internal/transport/middleware"

	"github.com/go-chi/chi"
)

func NewRouter(delivery *DeliveryHandler, event *EventHandler, webhook *WebhookHandler, store *middleware.IdempotencyStore) http.Handler {
	r := chi.NewRouter()

	r.Group(func(r chi.Router) {
		r.Use(middleware.Idempotency(store))
		r.Post("/events", event.CreateEvent)
		r.Post("/webhooks", webhook.CreateWebhook)
	})

	r.Get("/events", event.ListEvents)

	r.Get("/webhooks/{id}", webhook.GetByID)
	r.Put("/webhooks/{id}", webhook.UpdateWebhook)
	r.Delete("/webhooks/{id}", webhook.Delete)

	r.Get("/deliveries/{id}", delivery.GetByID)
	r.Get("/webhooks/{id}/deliveries", delivery.ListByWebhook)
	r.Get("/deliveries/{id}/attempts", delivery.GetAttempts)

	return r
}
