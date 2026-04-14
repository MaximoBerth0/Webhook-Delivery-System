package http

import (
	"encoding/json"
	"net/http"
	"webhook-delivery-system/internal/attempt"
	"webhook-delivery-system/internal/delivery"

	"github.com/go-chi/chi/v5"
)

type DeliveryHandler struct {
	deliveryService *delivery.Service
	attemptService  *attempt.Service
}

func NewDeliveryHandler(deliveryService *delivery.Service, attemptService *attempt.Service) *DeliveryHandler {
	return &DeliveryHandler{
		deliveryService: deliveryService,
		attemptService:  attemptService,
	}
}

func (h *DeliveryHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	result, err := h.deliveryService.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "delivery not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *DeliveryHandler) ListByWebhook(w http.ResponseWriter, r *http.Request) {
	webhookID := chi.URLParam(r, "id")

	results, err := h.deliveryService.GetByWebhookID(r.Context(), webhookID)
	if err != nil {
		http.Error(w, "could not fetch deliveries", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (h *DeliveryHandler) GetAttempts(w http.ResponseWriter, r *http.Request) {
	deliveryID := chi.URLParam(r, "id")

	results, err := h.attemptService.GetAttempts(r.Context(), deliveryID)
	if err != nil {
		http.Error(w, "could not fetch attempts", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}
