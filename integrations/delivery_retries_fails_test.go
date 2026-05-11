package integrations

import (
	"context"
	"net/http"
	"testing"
	"time"

	"net/http/httptest"
	"webhook-delivery-system/integrations/helpers"
	"webhook-delivery-system/internal/event"
	"webhook-delivery-system/internal/webhook"
)

func TestDeliveryRetriesFails(t *testing.T) {
	env := helpers.SetupEnv(t)
	ctx := context.Background()

	requestCount := 0

	// mock webhook that always fails with 500 internal server error
	mockWebhook := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		t.Logf("Attempt #%d received at mock webhook", requestCount)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer mockWebhook.Close()

	// create webhook with 3 max attempts
	webhook, err := env.WebhookSvc.CreateWebhook(ctx, webhook.CreateWebhookRequest{
		TargetURL:        mockWebhook.URL,
		Secret:           "test-secret",
		MaxAttempts:      3,
		SubscribedEvents: []event.SubscribedEvent{event.EventCustomerCreated},
	})
	if err != nil {
		t.Fatalf("create webhook: %v", err)
	}

	payload := []byte(`{"email": "test@example.com", "customer_id": "cust_123"}`)

	// create the event
	createdEvent, err := env.EventSvc.CreateEvent(ctx, event.CreateEventRequest{
		Type:    event.EventCustomerCreated,
		Payload: payload,
	})
	if err != nil {
		t.Fatalf("create event: %v", err)
	}

	// create the delivery
	_, err = env.DeliverySvc.Create(ctx, createdEvent.ID, webhook.ID)
	if err != nil {
		t.Fatalf("create delivery: %v", err)
	}

	// attempt 1: delivery is still PENDING, so fetch it and kick off the first attempt
	t.Log("Processing attempt #1 (first attempt)")
	pending, err := env.DeliverySvc.GetPending(ctx, 1)
	if err != nil {
		t.Fatalf("get pending deliveries: %v", err)
	}
	if len(pending) == 0 {
		t.Fatal("expected a pending delivery, got none")
	}
	if err := env.DeliveryWorker.ProcessFirstAttempt(ctx, &pending[0]); err != nil {
		t.Fatalf("ProcessFirstAttempt failed: %v", err)
	}

	// attempts 2 and 3: the worker scheduled retries, wait for them to become due then process
	for i := 2; i <= 3; i++ {
		time.Sleep(2 * time.Second)
		t.Logf("Processing attempt #%d (scheduled retry)", i)
		if err := env.DeliveryWorker.ProcessScheduledAttempts(ctx); err != nil {
			t.Fatalf("ProcessScheduledAttempts #%d failed: %v", i, err)
		}
	}

	// verify the final delivery state
	deliveries, err := env.DeliverySvc.GetByWebhookID(ctx, webhook.ID)
	if err != nil {
		t.Fatalf("get deliveries by webhook: %v", err)
	}

	if len(deliveries) != 1 {
		t.Fatalf("expected 1 delivery for webhook %s, got %d", webhook.ID, len(deliveries))
	}

	delivery := deliveries[0]

	// verify delivery ultimately failed after exhausting retries
	if delivery.Status != "FAILED" {
		t.Errorf("expected status 'FAILED' after all retries exhausted, got '%s'", delivery.Status)
	}

	// verify correct event association
	if delivery.EventID != createdEvent.ID {
		t.Errorf("expected event_id %s, got %s", createdEvent.ID, delivery.EventID)
	}

	// verify all retry attempts were made
	if requestCount != 3 {
		t.Errorf("expected exactly %d HTTP requests (max_attempts), got %d", 3, requestCount)
	}

	t.Logf("Success: Event %s → Delivery to webhook %s → Failed after %d attempts → Status: %s",
		createdEvent.ID, webhook.ID, requestCount, delivery.Status)
}
