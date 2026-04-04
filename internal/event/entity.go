package event

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type SubscribedEvent string

const (
	// customer
	EventCustomerCreated SubscribedEvent = "customer.created"
	EventCustomerUpdated SubscribedEvent = "customer.updated"
	EventCustomerDeleted SubscribedEvent = "customer.deleted"

	//payment
	EventPaymentCreated   SubscribedEvent = "payment.created"
	EventPaymentCompleted SubscribedEvent = "payment.completed"
	EventPaymentFailed    SubscribedEvent = "payment.failed"
	EventPaymentRefunded  SubscribedEvent = "payment.refunded"
	EventPaymentCancelled SubscribedEvent = "payment.cancelled"

	//subscription
	EventSubscriptionCreated   SubscribedEvent = "subscription.created"
	EventSubscriptionRenewed   SubscribedEvent = "subscription.renewed"
	EventSubscriptionCancelled SubscribedEvent = "subscription.cancelled"
	EventSubscriptionPastDue   SubscribedEvent = "subscription.past_due"
)

type Event struct {
	ID        string
	Type      SubscribedEvent
	Payload   []byte
	CreatedAt time.Time
}

func NewEvent(eventType SubscribedEvent, payload []byte) (*Event, error) {
	if !IsValidEvent(eventType) {
		return nil, errors.New("invalid event type")
	}
	if len(payload) == 0 {
		return nil, errors.New("payload is required")
	}
	return &Event{
		ID:        uuid.NewString(),
		Type:      eventType,
		Payload:   payload,
		CreatedAt: time.Now().UTC(),
	}, nil
}

func IsValidEvent(e SubscribedEvent) bool {
	switch e {
	case
		EventCustomerCreated, EventCustomerUpdated, EventCustomerDeleted,
		EventPaymentCreated, EventPaymentCompleted, EventPaymentFailed,
		EventPaymentRefunded, EventPaymentCancelled,
		EventSubscriptionCreated, EventSubscriptionRenewed,
		EventSubscriptionCancelled, EventSubscriptionPastDue:
		return true
	default:
		return false
	}
}
