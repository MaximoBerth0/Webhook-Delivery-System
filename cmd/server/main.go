package main

import (
	"context"
	"net/http"
	"os"

	"webhook-delivery-system/internal/attempt"
	"webhook-delivery-system/internal/delivery"
	"webhook-delivery-system/internal/event"
	"webhook-delivery-system/internal/infrastructure"
	"webhook-delivery-system/internal/storage"
	"webhook-delivery-system/internal/storage/postgres"
	transporthttp "webhook-delivery-system/internal/transport/http"
	"webhook-delivery-system/internal/webhook"
)

func main() {
	ctx := context.Background()

	log := infrastructure.NewLog()

	shutdown, err := infrastructure.SetupTelemetry(ctx, "webhook-delivery-system")
	if err != nil {
		log.Error("telemetry setup failed", "error", err)
	} else {
		defer shutdown(ctx)
	}

	pool, err := storage.NewPool(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Error("db connect", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	// repositories
	wRepo := postgres.NewWebhookRepository(pool)
	eRepo := postgres.NewEventRepository(pool)
	dRepo := postgres.NewDeliveryRepository(pool)
	aRepo := postgres.NewAttemptRepository(pool)

	idGen := infrastructure.NewUUIDGenerator()

	// services
	wSvc := webhook.NewService(wRepo, idGen, log)
	eSvc := event.NewService(eRepo, idGen, log)
	dSvc := delivery.NewService(dRepo, idGen, log)
	aSvc := attempt.NewService(aRepo, idGen, log)

	// handlers
	eventHandler := transporthttp.NewEventHandler(eSvc)
	deliveryHandler := transporthttp.NewDeliveryHandler(dSvc, aSvc)
	webhookHandler := transporthttp.NewWebhookHandler(wSvc)

	// router
	router := transporthttp.NewRouter(deliveryHandler, eventHandler, webhookHandler)

	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}

	log.Info("listening", "addr", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Error("server", "error", err)
		os.Exit(1)
	}
}
