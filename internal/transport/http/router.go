package http

import (
	"net/http"
	"webhook-delivery-system/internal/transport/middleware"

	"github.com/go-chi/chi"
)

func NewRouter(delivery *DeliveryHandler, event *EventHandler, webhook *WebhookHandler,
	store *middleware.IdempotencyStore) http.Handler {

	r := chi.NewRouter()

	// different rate limiters
	strictLimiter := middleware.NewRateLimiter(5, 10)    // 5 req/s, burst 10
	moderateLimiter := middleware.NewRateLimiter(10, 20) // 10 req/s, burst 20
	lenientLimiter := middleware.NewRateLimiter(50, 100) // 50 req/s, burst 100

	r.Group(func(r chi.Router) {
		r.Use(strictLimiter.Limit())
		r.Use(middleware.Idempotency(store))
		r.Post("/events", event.CreateEvent)
		r.Post("/webhooks", webhook.CreateWebhook)
	})

	r.Group(func(r chi.Router) {
		r.Use(moderateLimiter.Limit())
		r.Put("/webhooks/{id}", webhook.UpdateWebhook)
		r.Delete("/webhooks/{id}", webhook.Delete)
	})

	r.Group(func(r chi.Router) {
		r.Use(lenientLimiter.Limit())
		r.Get("/events", event.ListEvents)
		r.Get("/webhooks/{id}", webhook.GetByID)
		r.Get("/deliveries/{id}", delivery.GetByID)
		r.Get("/webhooks/{id}/deliveries", delivery.ListByWebhook)
		r.Get("/deliveries/{id}/attempts", delivery.GetAttempts)
	})

	return r
}
