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

func (h *Handler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Type    event.SubscribedEvent `json:"type"`
		Payload json.RawMessage       `json:"payload"` // JSON crudo
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	req := event.CreateEventRequest{
		Type:    body.Type,
		Payload: []byte(body.Payload),
	}

	if err := h.service.CreateEvent(r.Context(), req); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "event created"})
}
