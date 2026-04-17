package helpers

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"webhook-delivery-system/internal/attempt"
	"webhook-delivery-system/internal/delivery"
	"webhook-delivery-system/internal/event"
	"webhook-delivery-system/internal/storage"
	pgstore "webhook-delivery-system/internal/storage/postgres"
	transporthttp "webhook-delivery-system/internal/transport/http"
	"webhook-delivery-system/internal/webhook"
)

type TestEnv struct {
	Server      *httptest.Server
	Pool        *pgxpool.Pool
	DeliverySvc *delivery.Service
	WebhookSvc  *webhook.Service
	EventSvc    *event.Service
}

func SetupEnv(t *testing.T) *TestEnv {
	t.Helper()
	ctx := context.Background()

	// spin up real postgres container
	pgContainer, err := postgres.Run(ctx,
		"postgres:16",
		postgres.WithDatabase("webhooks_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForListeningPort("5432/tcp"),
		),
	)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}
	t.Cleanup(func() { pgContainer.Terminate(ctx) })

	//get connection string + open pool = same as main.go
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}
	pool, err := storage.NewPool(ctx, connStr)
	if err != nil {
		t.Fatalf("db connect: %v", err)
	}
	t.Cleanup(func() { pool.Close() })

	//run migrations against the test DB
	runMigrations(t, connStr)

	wRepo := pgstore.NewWebhookRepository(pool)
	eRepo := pgstore.NewEventRepository(pool)
	dRepo := pgstore.NewDeliveryRepository(pool)
	aRepo := pgstore.NewAttemptRepository(pool)

	wSvc := webhook.NewService(wRepo)
	eSvc := event.NewService(eRepo)
	dSvc := delivery.NewService(dRepo)
	aSvc := attempt.NewService(aRepo)

	eventHandler := transporthttp.NewEventHandler(eSvc)
	deliveryHandler := transporthttp.NewDeliveryHandler(dSvc, aSvc)
	webhookHandler := transporthttp.NewWebhookHandler(wSvc)

	router := transporthttp.NewRouter(deliveryHandler, eventHandler, webhookHandler)

	//real HTTP server on a random port
	server := httptest.NewServer(router)
	t.Cleanup(func() { server.Close() })

	return &TestEnv{
		Server:      server,
		Pool:        pool,
		DeliverySvc: dSvc,
		WebhookSvc:  wSvc,
		EventSvc:    eSvc,
	}
}
