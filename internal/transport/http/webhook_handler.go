package http

import (
	"encoding/json"
	"net/http"
	"webhook-delivery-system/internal/event"
	"webhook-delivery-system/internal/webhook"

	"github.com/go-chi/chi"
)

// webhook_handler.go — 4 HTTP functions
/*

webhook service:

func (s *Service) CreateWebhook(ctx context.Context, req CreateWebhookRequest) error {
func (s *Service) UpdateWebhook(ctx context.Context, id string, req CreateWebhookRequest) error {
func (s *Service) GetWebhook(ctx context.Context, id string) (*Webhook, error) {
func (s *Service) ListWebhooksByEvent(ctx context.Context, e event.SubscribedEvent) ([]Webhook, error) {
func (s *Service) DeleteWebhook(ctx context.Context, id string) error {


func (h *WebhookHandler) Create(w http.ResponseWriter, r *http.Request)
func (h *WebhookHandler) GetByID(w http.ResponseWriter, r *http.Request)
func (h *WebhookHandler) Update(w http.ResponseWriter, r *http.Request)
func (h *WebhookHandler) Delete(w http.ResponseWriter, r *http.Request)

*/

type WebhookHandler struct {
	service *webhook.Service
}

func NewWebhookHandler(service *webhook.Service) *WebhookHandler {
	return &WebhookHandler{service: service}
}

func (h *WebhookHandler) CreateWebhook(w http.ResponseWriter, r *http.Request) {
	var body struct {
		TargetURL        string                  `json:"target_url"`
		Secret           string                  `json:"secret"`
		MaxAttempts      int                     `json:"max_attempts"`
		SubscribedEvents []event.SubscribedEvent `json:"subscribed_events"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	req := webhook.CreateWebhookRequest{
		TargetURL:        body.TargetURL,
		Secret:           body.Secret,
		MaxAttempts:      body.MaxAttempts,
		SubscribedEvents: body.SubscribedEvents,
	}

	if _, err := h.service.CreateWebhook(r.Context(), req); err != nil {
		http.Error(w, "failed to create webhook", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *WebhookHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	result, err := h.service.GetWebhook(r.Context(), id)
	if err != nil {
		http.Error(w, "webhook not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *WebhookHandler) UpdateWebhook(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var body struct {
		TargetURL        string                  `json:"target_url"`
		Secret           string                  `json:"secret"`
		MaxAttempts      int                     `json:"max_attempts"`
		SubscribedEvents []event.SubscribedEvent `json:"subscribed_events"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	req := webhook.CreateWebhookRequest{
		TargetURL:        body.TargetURL,
		Secret:           body.Secret,
		MaxAttempts:      body.MaxAttempts,
		SubscribedEvents: body.SubscribedEvents,
	}

	if err := h.service.UpdateWebhook(r.Context(), id, req); err != nil {
		http.Error(w, "failed to update webhook", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *WebhookHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	err := h.service.DeleteWebhook(r.Context(), id)
	if err != nil {
		http.Error(w, "webhook not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)

}
