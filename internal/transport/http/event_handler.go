package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"webhook-delivery-system/internal/event"
)

type EventHandler struct {
	service *event.Service
}

const defaultLimit = 20
const maxLimit = 100

func NewEventHandler(service *event.Service) *EventHandler {
	return &EventHandler{service: service}
}

func (h *EventHandler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Type    event.SubscribedEvent `json:"type"`
		Payload json.RawMessage       `json:"payload"`
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

func (h *EventHandler) ListEvents(w http.ResponseWriter, r *http.Request) {
	limit := defaultLimit
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
			if limit > maxLimit {
				limit = maxLimit
			}
		}
	}

	events, err := h.service.ListEvents(r.Context(), limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
}
