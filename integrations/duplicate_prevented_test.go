package integrations

import (
	"context"
	"net/http"
	"testing"

	"net/http/httptest"
	"webhook-delivery-system/integrations/helpers"
	"webhook-delivery-system/internal/event"
	"webhook-delivery-system/internal/webhook"
)

func TestDuplicatePrevented(t *testing.T) {
	env := helpers.SetupEnv(t)
	ctx := context.Background()

	requestCount := 0
	mockWebhook := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer mockWebhook.Close()

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
	createdEvent, err := env.EventSvc.CreateEvent(ctx, event.CreateEventRequest{
		Type:    event.EventCustomerCreated,
		Payload: payload,
	})
	if err != nil {
		t.Fatalf("create event: %v", err)
	}

	// create first delivery - should succeed
	delivery1, err := env.DeliverySvc.Create(ctx, createdEvent.ID, webhook.ID)
	if err != nil {
		t.Fatalf("create first delivery: %v", err)
	}

	//try to create duplicate - should fail
	_, err = env.DeliverySvc.Create(ctx, createdEvent.ID, webhook.ID)
	if err == nil {
		t.Fatalf("expected error when creating duplicate delivery, got nil")
	}
	// verify only ONE delivery exists
	deliveries, err := env.DeliverySvc.GetByWebhookID(ctx, webhook.ID)
	if err != nil {
		t.Fatalf("get deliveries: %v", err)
	}
	if len(deliveries) != 1 {
		t.Errorf("expected exactly 1 delivery, got %d", len(deliveries))
	}
	if len(deliveries) > 0 && deliveries[0].ID != delivery1.ID {
		t.Errorf("expected delivery ID %s, got %s", delivery1.ID, deliveries[0].ID)
	}

	t.Logf("Duplicate prevention working: only 1 delivery exists")
}
