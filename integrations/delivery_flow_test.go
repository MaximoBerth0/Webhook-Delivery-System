package integrations

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"webhook-delivery-system/integrations/helpers"
	"webhook-delivery-system/internal/delivery"
	"webhook-delivery-system/internal/event"
	"webhook-delivery-system/internal/webhook"
)

func seedWebhookAndEvent(t *testing.T, env *helpers.TestEnv) (webhookID, eventID string) {
	t.Helper()
	ctx := context.Background()

	wh, err := env.WebhookSvc.CreateWebhook(ctx, webhook.CreateWebhookRequest{
		TargetURL:        "https://example.com/hook",
		Secret:           "secret124244",
		MaxAttempts:      3,
		SubscribedEvents: []event.SubscribedEvent{event.EventCustomerCreated},
	})
	if err != nil {
		t.Fatalf("seed webhook: %v", err)
	}

	ev, err := env.EventSvc.CreateEvent(ctx, event.CreateEventRequest{
		Type:    event.EventCustomerCreated,
		Payload: []byte(`{"id":"123"}`),
	})
	if err != nil {
		t.Fatalf("seed event: %v", err)
	}

	return wh.ID, ev.ID
}

func TestDelivery_GetByID(t *testing.T) {
	env := helpers.SetupEnv(t)
	webhookID, eventID := seedWebhookAndEvent(t, env)

	created, err := env.DeliverySvc.Create(context.Background(), eventID, webhookID)
	if err != nil {
		t.Fatalf("seed delivery: %v", err)
	}

	resp, err := http.Get(env.Server.URL + "/deliveries/" + created.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("want 200, got %d", resp.StatusCode)
	}

	var got delivery.Delivery
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("want ID %s, got %s", created.ID, got.ID)
	}
}

func TestDelivery_GetByID_NotFound(t *testing.T) {
	env := helpers.SetupEnv(t)

	resp, err := http.Get(env.Server.URL + "/deliveries/non-existent-id")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("want 404, got %d", resp.StatusCode)
	}
}

func TestDelivery_ListByWebhook(t *testing.T) {
	env := helpers.SetupEnv(t)
	webhookID, eventID1 := seedWebhookAndEvent(t, env)

	ev2, err := env.EventSvc.CreateEvent(context.Background(), event.CreateEventRequest{
		Type:    event.EventCustomerCreated,
		Payload: []byte(`{"id":"456"}`),
	})
	if err != nil {
		t.Fatalf("seed second event: %v", err)
	}

	env.DeliverySvc.Create(context.Background(), eventID1, webhookID)
	env.DeliverySvc.Create(context.Background(), ev2.ID, webhookID)

	resp, err := http.Get(env.Server.URL + "/webhooks/" + webhookID + "/deliveries")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("want 200, got %d", resp.StatusCode)
	}

	var got []delivery.Delivery
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("want 2 deliveries, got %d", len(got))
	}
}

func TestDelivery_GetAttempts_Empty(t *testing.T) {
	env := helpers.SetupEnv(t)
	webhookID, eventID := seedWebhookAndEvent(t, env)

	created, err := env.DeliverySvc.Create(context.Background(), eventID, webhookID)
	if err != nil {
		t.Fatalf("seed delivery: %v", err)
	}

	resp, err := http.Get(env.Server.URL + "/deliveries/" + created.ID + "/attempts")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("want 200, got %d", resp.StatusCode)
	}
}
