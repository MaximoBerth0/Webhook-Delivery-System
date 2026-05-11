package integrations

import (
	"context"
	"io"
	"net/http"
	"testing"

	"net/http/httptest"
	"webhook-delivery-system/integrations/helpers"
	"webhook-delivery-system/internal/event"
	"webhook-delivery-system/internal/webhook"
)

func TestFullDeliverySuccess(t *testing.T) {
	env := helpers.SetupEnv(t)
	ctx := context.Background()

	var receivedPayload []byte
	var receivedSignature string

	mockWebhook := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPayload, _ = io.ReadAll(r.Body)
		receivedSignature = r.Header.Get("X-Webhook-Signature")
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

	// process the delivery
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

	deliveries, err := env.DeliverySvc.GetByWebhookID(ctx, webhook.ID)
	if err != nil {
		t.Fatalf("get deliveries by webhook: %v", err)
	}

	if len(deliveries) != 1 {
		t.Fatalf("expected 1 delivery for webhook %s, got %d", webhook.ID, len(deliveries))
	}

	delivery := deliveries[0]

	if delivery.Status != "SUCCESS" {
		t.Errorf("expected status 'SUCCESS', got '%s'", delivery.Status)
	}

	if delivery.EventID != createdEvent.ID {
		t.Errorf("expected event_id %s, got %s", createdEvent.ID, delivery.EventID)
	}

	if receivedPayload == nil {
		t.Fatal("mock webhook never received a request")
	}

	if string(receivedPayload) != string(payload) {
		t.Errorf("payload mismatch:\nwant: %s\ngot:  %s",
			string(payload),
			string(receivedPayload))
	}

	if receivedSignature == "" {
		t.Error("expected X-Webhook-Signature header, got empty")
	}

	t.Logf("Success: Event %s → Delivery to webhook %s → Status: %s",
		createdEvent.ID, webhook.ID, delivery.Status)
}
