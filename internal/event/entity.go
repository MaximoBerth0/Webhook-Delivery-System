package event

import "time"

type SubscribedEvent string

const (
	EventUserCreated SubscribedEvent = "user.created"
	EventUserUpdated SubscribedEvent = "user.updated"
	EventUserDeleted SubscribedEvent = "user.deleted"
)

type Event struct {
	ID        string
	Type      SubscribedEvent
	Payload   []byte
	CreatedAt time.Time
}

func IsValidEvent(e SubscribedEvent) bool {
	switch e {
	case EventUserCreated, EventUserUpdated, EventUserDeleted:
		return true
	default:
		return false
	}
}
