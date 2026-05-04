package event

// TestNewEvent
// TestIsValidEvent

import (
	"testing"
)

func TestNewEvent(t *testing.T) {
	validPayload := []byte(`{"key":"value"}`)

	tests := []struct {
		name      string
		eventType SubscribedEvent
		payload   []byte
		wantErr   error
	}{
		{"valid customer event", EventCustomerCreated, validPayload, nil},
		{"valid payment event", EventPaymentCompleted, validPayload, nil},
		{"valid subscription event", EventSubscriptionCreated, validPayload, nil},
		{"invalid event type", "unknown.event", validPayload, ErrInvalidEventType},
		{"empty event type", "", validPayload, ErrInvalidEventType},
		{"empty payload", EventCustomerCreated, []byte{}, ErrInvalidPayload},
		{"nil payload", EventCustomerCreated, nil, ErrInvalidPayload},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewEvent(tt.eventType, tt.payload)
			if err != tt.wantErr {
				t.Errorf("NewEvent() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr == nil {
				if got == nil {
					t.Fatal("NewEvent() returned nil event with no error")
				}
				if got.Type != tt.eventType {
					t.Errorf("NewEvent() Type = %v, want %v", got.Type, tt.eventType)
				}
				if string(got.Payload) != string(tt.payload) {
					t.Errorf("NewEvent() Payload = %v, want %v", got.Payload, tt.payload)
				}
				if got.CreatedAt.IsZero() {
					t.Error("NewEvent() CreatedAt should not be zero")
				}
			}
		})
	}
}

func TestIsValidEvent(t *testing.T) {
	validEvents := []SubscribedEvent{
		EventCustomerCreated, EventCustomerUpdated, EventCustomerDeleted,
		EventPaymentCreated, EventPaymentCompleted, EventPaymentFailed,
		EventPaymentRefunded, EventPaymentCancelled,
		EventSubscriptionCreated, EventSubscriptionRenewed,
		EventSubscriptionCancelled, EventSubscriptionPastDue,
	}

	for _, e := range validEvents {
		t.Run(string(e), func(t *testing.T) {
			if !IsValidEvent(e) {
				t.Errorf("IsValidEvent(%q) = false, want true", e)
			}
		})
	}

	invalidEvents := []SubscribedEvent{
		"",
		"unknown.event",
		"customer",
		"payment",
		"customer.created.extra",
	}

	for _, e := range invalidEvents {
		t.Run("invalid:"+string(e), func(t *testing.T) {
			if IsValidEvent(e) {
				t.Errorf("IsValidEvent(%q) = true, want false", e)
			}
		})
	}
}
