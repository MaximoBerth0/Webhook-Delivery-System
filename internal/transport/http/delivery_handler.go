package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"webhook-delivery-system/internal/attempt"
	"webhook-delivery-system/internal/delivery"
	"webhook-delivery-system/internal/webhook"

	"log/slog"

	"github.com/go-chi/chi"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type DeliveryHandler struct {
	deliveryService *delivery.Service
	attemptService  *attempt.Service
	logger          *slog.Logger
	tracer          trace.Tracer
}

func NewDeliveryHandler(deliveryService *delivery.Service, attemptService *attempt.Service, logger *slog.Logger) *DeliveryHandler {
	return &DeliveryHandler{
		deliveryService: deliveryService,
		attemptService:  attemptService,
		logger:          logger,
		tracer:          otel.Tracer("http.delivery_handler"),
	}
}

func (h *DeliveryHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "DeliveryHandler.GetByID")
	defer span.End()

	id := chi.URLParam(r, "id")
	span.SetAttributes(attribute.String("delivery.id", id))

	result, err := h.deliveryService.GetByID(ctx, id)
	if err != nil {

		if errors.Is(err, delivery.ErrDeliveryNotFound) {
			span.SetStatus(codes.Error, "delivery not found")
			span.RecordError(err)
			http.Error(w, "delivery not found", http.StatusNotFound)
			return
		}

		span.SetStatus(codes.Error, "internal server error")
		span.RecordError(err)

		h.logger.Error("failed to get delivery",
			slog.String("delivery_id", id),
			slog.String("error", err.Error()),
			slog.String("trace_id", getTraceID(ctx)),
		)

		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	span.SetStatus(codes.Ok, "delivery retrieved")

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		h.logger.Error("failed to encode response",
			slog.String("error", err.Error()),
			slog.String("trace_id", getTraceID(ctx)),
		)
	}
}

func (h *DeliveryHandler) ListByWebhook(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "DeliveryHandler.ListByWebhook")
	defer span.End()

	webhookID := chi.URLParam(r, "id")
	span.SetAttributes(attribute.String("webhook.id", webhookID))

	deliveries, err := h.deliveryService.GetByWebhookID(ctx, webhookID)
	if err != nil {
		if errors.Is(err, webhook.ErrWebhookNotFound) {
			span.SetStatus(codes.Error, "webhook not found")
			span.RecordError(err)
			http.Error(w, "webhook not found", http.StatusNotFound)
			return
		}

		span.SetStatus(codes.Error, "internal server error")
		span.RecordError(err)

		h.logger.Error("failed to list deliveries",
			slog.String("webhook_id", webhookID),
			slog.String("error", err.Error()),
			slog.String("trace_id", getTraceID(ctx)),
		)

		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	span.SetAttributes(attribute.Int("deliveries.count", len(deliveries)))
	span.SetStatus(codes.Ok, "deliveries retrieved")

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(deliveries); err != nil {
		h.logger.Error("failed to encode response",
			slog.String("error", err.Error()),
			slog.String("trace_id", getTraceID(ctx)),
		)
	}
}

func (h *DeliveryHandler) GetAttempts(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "DeliveryHandler.GetAttempts")
	defer span.End()

	deliveryID := chi.URLParam(r, "id")
	span.SetAttributes(attribute.String("delivery_id", deliveryID))

	attempts, err := h.attemptService.GetAttempts(ctx, deliveryID)
	if err != nil {
		span.SetStatus(codes.Error, "internal server error")
		span.RecordError(err)

		h.logger.Error("failed to get delivery attempts",
			slog.String("delivery_id", deliveryID),
			slog.String("error", err.Error()),
			slog.String("trace_id", getTraceID(ctx)),
		)

		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	span.SetAttributes(attribute.Int("attempts.count", len(attempts)))
	span.SetStatus(codes.Ok, "attempts retrieved")

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(attempts); err != nil {
		h.logger.Error("failed to encode response",
			slog.String("error", err.Error()),
			slog.String("trace_id", getTraceID(ctx)),
		)
	}
}
