package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"webhook-delivery-system/internal/attempt"
	"webhook-delivery-system/internal/delivery"
	"webhook-delivery-system/internal/event"
	"webhook-delivery-system/internal/infrastructure"
	"webhook-delivery-system/internal/infrastructure/config"
	"webhook-delivery-system/internal/storage"
	"webhook-delivery-system/internal/storage/postgres"
	transporthttp "webhook-delivery-system/internal/transport/http"
	"webhook-delivery-system/internal/transport/middleware"
	"webhook-delivery-system/internal/webhook"
	"webhook-delivery-system/internal/worker"
)

func main() {
	ctx := context.Background()

	// load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		os.Exit(1)
	}

	// setup logger
	logger := infrastructure.NewLog()
	logger.Info("starting application",
		slog.String("service", cfg.Telemetry.ServiceName),
		slog.String("version", cfg.Telemetry.Version),
		slog.String("environment", cfg.Telemetry.Environment),
	)

	// setup telemetry
	shutdown, err := infrastructure.SetupTelemetry(ctx)
	if err != nil {
		logger.Error("failed to setup telemetry",
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}
	defer shutdown(ctx)

	// database connection
	pool, err := storage.NewPool(ctx, cfg.Database.URL)
	if err != nil {
		logger.Error("failed to connect to database",
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}
	defer pool.Close()

	// Configure connection pool
	pool.Config().MaxConns = int32(cfg.Database.MaxConns)
	pool.Config().MinConns = int32(cfg.Database.MinConns)

	logger.Info("database connected",
		slog.Int("max_conns", cfg.Database.MaxConns),
		slog.Int("min_conns", cfg.Database.MinConns),
	)

	// repositories
	wRepo := postgres.NewWebhookRepository(pool, logger)
	eRepo := postgres.NewEventRepository(pool, logger)
	dRepo := postgres.NewDeliveryRepository(pool, logger)
	aRepo := postgres.NewAttemptRepository(pool, logger)

	// ID generator
	idGen := infrastructure.NewUUIDGenerator()

	// services
	wSvc := webhook.NewService(wRepo, idGen, logger)
	eSvc := event.NewService(eRepo, idGen, logger)
	dSvc := delivery.NewService(dRepo, idGen, logger)
	aSvc := attempt.NewService(aRepo, idGen, logger)

	// signer for webhooks
	signer := infrastructure.NewHMACSigner()

	// delivery worker
	deliveryWorker := worker.NewDeliveryWorker(
		dSvc,
		wRepo,
		eRepo,
		aSvc,
		signer,
		logger,
	)

	// dispatcher manages multiple workers
	dispatcher := worker.NewDispatcher(
		dSvc,
		deliveryWorker,
		logger,
		cfg.Worker,
	)

	// start dispatcher
	logger.Info("starting delivery workers",
		slog.Int("concurrency", cfg.Worker.Concurrency),
		slog.Duration("poll_interval", cfg.Worker.PollInterval),
		slog.Int("batch_size", cfg.Worker.BatchSize),
	)
	dispatcher.Run(ctx)

	// HTTP handlers
	eventHandler := transporthttp.NewEventHandler(eSvc, logger)
	deliveryHandler := transporthttp.NewDeliveryHandler(dSvc, aSvc, logger)
	webhookHandler := transporthttp.NewWebhookHandler(wSvc, logger)

	// router
	idempotencyStore := middleware.NewIdempotencyStore(logger)
	router := transporthttp.NewRouter(
		deliveryHandler,
		eventHandler,
		webhookHandler,
		idempotencyStore,
	)

	// HTTP Server
	server := &http.Server{
		Addr:         cfg.Server.Addr,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// start server in goroutine
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("http server listening", slog.String("addr", cfg.Server.Addr))
		serverErrors <- server.ListenAndServe()
	}()

	// wait for interrupt signal or server error
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		logger.Error("server error", slog.String("error", err.Error()))
		os.Exit(1)

	case sig := <-quit:
		logger.Info("shutdown signal received",
			slog.String("signal", sig.String()),
		)

		shutdownCtx, cancel := context.WithTimeout(ctx, cfg.Server.ShutdownTimeout)
		defer cancel()

		// shutdown HTTP server
		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("graceful shutdown failed",
				slog.String("error", err.Error()),
			)
			if err := server.Close(); err != nil {
				logger.Error("force shutdown failed",
					slog.String("error", err.Error()),
				)
			}
		}

		logger.Info("shutdown complete")

		// stop dispatcher (waits for all workers)
		dispatcher.Stop()
		logger.Info("all workers stopped")
	}
}
