package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"webhook-delivery-system/internal/event"
	"webhook-delivery-system/internal/webhook"

	"github.com/go-chi/chi"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type WebhookHandler struct {
	service *webhook.Service
	logger  *slog.Logger
	tracer  trace.Tracer
}

func NewWebhookHandler(service *webhook.Service, logger *slog.Logger) *WebhookHandler {
	return &WebhookHandler{
		service: service,
		logger:  logger,
		tracer:  otel.Tracer("http.webhook_handler"),
	}
}

func (h *WebhookHandler) CreateWebhook(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "WebhookHandler.CreateWebhook")
	defer span.End()
	defer r.Body.Close()

	var body struct {
		TargetURL        string                  `json:"target_url"`
		Secret           string                  `json:"secret"`
		MaxAttempts      int                     `json:"max_attempts"`
		SubscribedEvents []event.SubscribedEvent `json:"subscribed_events"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		span.SetStatus(codes.Error, "invalid request body")
		span.RecordError(err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	span.SetAttributes(
		attribute.String("webhook.target_url", body.TargetURL),
		attribute.Int("webhook.max_attempts", body.MaxAttempts),
	)

	req := webhook.CreateWebhookRequest{
		TargetURL:        body.TargetURL,
		Secret:           body.Secret,
		MaxAttempts:      body.MaxAttempts,
		SubscribedEvents: body.SubscribedEvents,
	}

	created, err := h.service.CreateWebhook(ctx, req)
	if err != nil {
		if errors.Is(err, webhook.ErrURLrequired) || errors.Is(err, webhook.ErrInvalidTargetURL) ||
			errors.Is(err, webhook.ErrSecretRequired) || errors.Is(err, webhook.ErrInvalidMaxAttempts) ||
			errors.Is(err, webhook.ErrNumberOfEvents) || errors.Is(err, webhook.ErrInvalidEventType) {
			span.SetStatus(codes.Error, err.Error())
			span.RecordError(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		span.SetStatus(codes.Error, "internal server error")
		span.RecordError(err)
		h.logger.Error("failed to create webhook",
			slog.String("error", err.Error()),
			slog.String("trace_id", getTraceID(ctx)),
		)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	span.SetAttributes(attribute.String("webhook.id", created.ID))
	span.SetStatus(codes.Ok, "webhook created")

	w.WriteHeader(http.StatusCreated)
}

func (h *WebhookHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "WebhookHandler.GetByID")
	defer span.End()

	id := chi.URLParam(r, "id")
	span.SetAttributes(attribute.String("webhook.id", id))

	result, err := h.service.GetWebhook(ctx, id)
	if err != nil {
		if errors.Is(err, webhook.ErrGetWebhook) {
			span.SetStatus(codes.Error, "webhook not found")
			span.RecordError(err)
			http.Error(w, "webhook not found", http.StatusNotFound)
			return
		}

		span.SetStatus(codes.Error, "internal server error")
		span.RecordError(err)
		h.logger.Error("failed to get webhook",
			slog.String("webhook_id", id),
			slog.String("error", err.Error()),
			slog.String("trace_id", getTraceID(ctx)),
		)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	span.SetStatus(codes.Ok, "webhook retrieved")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *WebhookHandler) UpdateWebhook(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "WebhookHandler.UpdateWebhook")
	defer span.End()
	defer r.Body.Close()

	id := chi.URLParam(r, "id")
	span.SetAttributes(attribute.String("webhook.id", id))

	var body struct {
		TargetURL        string                  `json:"target_url"`
		Secret           string                  `json:"secret"`
		MaxAttempts      int                     `json:"max_attempts"`
		SubscribedEvents []event.SubscribedEvent `json:"subscribed_events"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		span.SetStatus(codes.Error, "invalid request body")
		span.RecordError(err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	req := webhook.CreateWebhookRequest{
		TargetURL:        body.TargetURL,
		Secret:           body.Secret,
		MaxAttempts:      body.MaxAttempts,
		SubscribedEvents: body.SubscribedEvents,
	}

	if err := h.service.UpdateWebhook(ctx, id, req); err != nil {
		if errors.Is(err, webhook.ErrURLrequired) || errors.Is(err, webhook.ErrInvalidTargetURL) ||
			errors.Is(err, webhook.ErrSecretRequired) || errors.Is(err, webhook.ErrInvalidMaxAttempts) ||
			errors.Is(err, webhook.ErrNumberOfEvents) || errors.Is(err, webhook.ErrInvalidEventType) {
			span.SetStatus(codes.Error, err.Error())
			span.RecordError(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		span.SetStatus(codes.Error, "internal server error")
		span.RecordError(err)
		h.logger.Error("failed to update webhook",
			slog.String("webhook_id", id),
			slog.String("error", err.Error()),
			slog.String("trace_id", getTraceID(ctx)),
		)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	span.SetStatus(codes.Ok, "webhook updated")
	w.WriteHeader(http.StatusOK)
}

func (h *WebhookHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "WebhookHandler.Delete")
	defer span.End()

	id := chi.URLParam(r, "id")
	span.SetAttributes(attribute.String("webhook.id", id))

	if err := h.service.DeleteWebhook(ctx, id); err != nil {
		span.SetStatus(codes.Error, "internal server error")
		span.RecordError(err)
		h.logger.Error("failed to delete webhook",
			slog.String("webhook_id", id),
			slog.String("error", err.Error()),
			slog.String("trace_id", getTraceID(ctx)),
		)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	span.SetStatus(codes.Ok, "webhook deleted")
	w.WriteHeader(http.StatusNoContent)
}
