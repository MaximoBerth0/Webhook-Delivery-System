package http

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"webhook-delivery-system/internal/event"
)

type EventHandler struct {
	service *event.Service
	logger  *slog.Logger
	tracer  trace.Tracer
}

const (
	defaultLimit = 20
	maxLimit     = 100
)

func NewEventHandler(service *event.Service, logger *slog.Logger) *EventHandler {
	return &EventHandler{
		service: service,
		logger:  logger,
		tracer:  otel.Tracer("http.event_handler"),
	}
}

func (h *EventHandler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "EventHandler.CreateEvent")
	defer span.End()

	var body struct {
		Type    event.SubscribedEvent `json:"type"`
		Payload json.RawMessage       `json:"payload"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		span.SetStatus(codes.Error, "invalid request body")
		span.RecordError(err)
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	span.SetAttributes(
		attribute.String("event.type", string(body.Type)),
		attribute.Int("payload.size", len(body.Payload)),
	)

	req := event.CreateEventRequest{
		Type:    body.Type,
		Payload: []byte(body.Payload),
	}

	createdEvent, err := h.service.CreateEvent(ctx, req)
	if err != nil {
		if errors.Is(err, event.ErrInvalidEventType) {
			span.SetStatus(codes.Error, "invalid event type")
			span.RecordError(err)
			http.Error(w, "invalid event type", http.StatusBadRequest)
			return
		}

		if errors.Is(err, event.ErrInvalidPayload) {
			span.SetStatus(codes.Error, "invalid payload")
			span.RecordError(err)
			http.Error(w, "invalid payload", http.StatusBadRequest)
			return
		}

		span.SetStatus(codes.Error, "internal server error")
		span.RecordError(err)

		h.logger.Error("failed to create event",
			slog.String("error", err.Error()),
			slog.String("event_type", string(body.Type)),
			slog.String("trace_id", getTraceID(ctx)),
		)

		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	span.SetAttributes(attribute.String("event.id", createdEvent.ID))
	span.SetStatus(codes.Ok, "event created")

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"status":   "event created",
		"event_id": createdEvent.ID,
	})
}

func (h *EventHandler) ListEvents(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "EventHandler.ListEvents")
	defer span.End()

	limit := defaultLimit
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
			if limit > maxLimit {
				limit = maxLimit
			}
		}
	}

	span.SetAttributes(attribute.Int("limit", limit))

	events, err := h.service.ListEvents(ctx, limit)
	if err != nil {
		span.SetStatus(codes.Error, "failed to list events")
		span.RecordError(err)

		h.logger.Error("failed to list events",
			slog.String("error", err.Error()),
			slog.Int("limit", limit),
			slog.String("trace_id", getTraceID(ctx)),
		)

		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	span.SetAttributes(attribute.Int("events.count", len(events)))
	span.SetStatus(codes.Ok, "events retrieved")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
}

// extract trace ID  (helper)
func getTraceID(ctx context.Context) string {
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.HasTraceID() {
		return spanCtx.TraceID().String()
	}
	return ""
}
