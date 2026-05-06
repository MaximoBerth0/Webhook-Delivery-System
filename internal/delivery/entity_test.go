package delivery

import "testing"

// run: go test ./internal/delivery/ -v

func TestNewDelivery(t *testing.T) {
	t.Run("valid inputs", func(t *testing.T) {
		d, err := NewDelivery("event-1", "webhook-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if d.Status != StatusPending {
			t.Errorf("Status = %q, want %q", d.Status, StatusPending)
		}
		if d.Attempts != 0 {
			t.Errorf("Attempts = %d, want 0", d.Attempts)
		}
	})

	t.Run("empty eventID", func(t *testing.T) {
		_, err := NewDelivery("", "webhook-1")
		if err != ErrInvalidEventID {
			t.Errorf("err = %v, want %v", err, ErrInvalidEventID)
		}
	})

	t.Run("empty webhookID", func(t *testing.T) {
		_, err := NewDelivery("event-1", "")
		if err != ErrInvalidWebhookID {
			t.Errorf("err = %v, want %v", err, ErrInvalidWebhookID)
		}
	})
}

func TestDeliveryStatusTransitions(t *testing.T) {
	tests := []struct {
		name       string
		transition func(*Delivery)
		want       Status
	}{
		{"MarkSuccess", (*Delivery).MarkSuccess, StatusSuccess},
		{"MarkRetry", (*Delivery).MarkRetry, StatusRetry},
		{"MarkFailed", (*Delivery).MarkFailed, StatusFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, _ := NewDelivery("event-1", "webhook-1")
			tt.transition(d)
			if d.Status != tt.want {
				t.Errorf("Status = %q, want %q", d.Status, tt.want)
			}
		})
	}
}

func TestRegisterAttempt(t *testing.T) {
	d, _ := NewDelivery("event-1", "webhook-1")

	for i := 1; i <= 3; i++ {
		d.RegisterAttempt()
		if d.Attempts != i {
			t.Errorf("after %d calls: Attempts = %d, want %d", i, d.Attempts, i)
		}
	}
}
