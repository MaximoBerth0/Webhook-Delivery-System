package http

import (
	"encoding/json"
	"net/http"

	"webhook-delivery-system/internal/event"
)

type Handler struct {
	service *event.Service
}

func NewHandler(service *event.Service) *Handler {
	return &Handler{service: service}
}

type CreateEventRequest struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

func (h *Handler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	err := h.service.CreateEvent(ctx, event.CreateEventRequest{
		Type:    req.Type,
		Payload: req.Payload,
	})

	if err != nil {
		http.Error(w, "failed to create event", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}
