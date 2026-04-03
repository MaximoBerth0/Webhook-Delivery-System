package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"webhook-delivery-system/internal/attempt"
	"webhook-delivery-system/internal/delivery"
	"webhook-delivery-system/internal/event"
	"webhook-delivery-system/internal/storage"
	"webhook-delivery-system/internal/storage/postgres"
	transporthttp "webhook-delivery-system/internal/transport/http"
	"webhook-delivery-system/internal/webhook"
)

func main() {
	ctx := context.Background()

	pool, err := storage.NewPool(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	// repositories
	wRepo := postgres.NewWebhookRepository(pool)
	eRepo := postgres.NewEventRepository(pool)
	dRepo := postgres.NewDeliveryRepository(pool)
	aRepo := postgres.NewAttemptRepository(pool)

	// services
	wSvc := webhook.NewService(wRepo)
	eSvc := event.NewService(eRepo)
	dSvc := delivery.NewService(dRepo)
	aSvc := attempt.NewService(aRepo)

	_ = wSvc
	_ = dSvc
	_ = aSvc

	// handlers
	eventHandler := transporthttp.NewHandler(eSvc)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /events", eventHandler.CreateEvent)

	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server: %v", err)
	}
}
