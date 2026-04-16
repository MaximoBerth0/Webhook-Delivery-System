package integrations

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"webhook-delivery-system/integrations/helpers"
	"webhook-delivery-system/internal/delivery"
)

func TestDelivery_GetByID(t *testing.T) {
	env := helpers.SetupEnv(t)

	created, err := env.DeliverySvc.Create(context.Background(), "event-id-1", "webhook-id-1")
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

	// seed two deliveries for the same webhook
	webhookID := "webhook-id-1"
	env.DeliverySvc.Create(context.Background(), "event-id-1", webhookID)
	env.DeliverySvc.Create(context.Background(), "event-id-2", webhookID)

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

	created, err := env.DeliverySvc.Create(context.Background(), "event-id-1", "webhook-id-1")
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
