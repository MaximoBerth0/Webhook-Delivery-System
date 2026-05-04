package webhook

// this test covers empty/invalid URL, empty secret, maxAttempt
// empty/nil events and invalid event type

import (
	"testing"
	"webhook-delivery-system/internal/event"
)

func TestNewWebhook(t *testing.T) {
	validEvents := []event.SubscribedEvent{event.EventCustomerCreated}

	tests := []struct {
		name        string
		targetURL   string
		secret      string
		maxAttempts int
		events      []event.SubscribedEvent
		wantErr     error
	}{
		{"valid webhook", "https://example.com/hook", "secret", 3, validEvents, nil},
		{"empty URL", "", "secret", 3, validEvents, ErrURLrequired},
		{"invalid URL", "not-a-url", "secret", 3, validEvents, ErrInvalidTargetURL},
		{"empty secret", "https://example.com/hook", "", 3, validEvents, ErrSecretRequired},
		{"zero max attempts", "https://example.com/hook", "secret", 0, validEvents, ErrInvalidMaxAttempts},
		{"negative max attempts", "https://example.com/hook", "secret", -1, validEvents, ErrInvalidMaxAttempts},
		{"no events", "https://example.com/hook", "secret", 3, []event.SubscribedEvent{}, ErrNumberOfEvents},
		{"nil events", "https://example.com/hook", "secret", 3, nil, ErrNumberOfEvents},
		{"invalid event type", "https://example.com/hook", "secret", 3, []event.SubscribedEvent{"unknown.event"}, ErrInvalidEventType},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewWebhook(tt.targetURL, tt.secret, tt.maxAttempts, tt.events)
			if err != tt.wantErr {
				t.Errorf("NewWebhook() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr == nil {
				if got == nil {
					t.Fatal("NewWebhook() returned nil with no error")
				}
				if got.TargetURL != tt.targetURL {
					t.Errorf("NewWebhook() TargetURL = %v, want %v", got.TargetURL, tt.targetURL)
				}
				if got.Secret != tt.secret {
					t.Errorf("NewWebhook() Secret = %v, want %v", got.Secret, tt.secret)
				}
				if got.MaxAttempts != tt.maxAttempts {
					t.Errorf("NewWebhook() MaxAttempts = %v, want %v", got.MaxAttempts, tt.maxAttempts)
				}
				if !got.Active {
					t.Error("NewWebhook() Active should be true")
				}
				if got.CreatedAt.IsZero() {
					t.Error("NewWebhook() CreatedAt should not be zero")
				}
				if got.UpdatedAt.IsZero() {
					t.Error("NewWebhook() UpdatedAt should not be zero")
				}
			}
		})
	}
}
