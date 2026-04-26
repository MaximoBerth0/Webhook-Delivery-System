package event

import "errors"

var (
	ErrInvalidEventType = errors.New("event: invalid event type")
	ErrInvalidPayload   = errors.New("event: payload cannot be empty")
	ErrCreateEvent      = errors.New("event: error creating event")
	ErrGetEvent         = errors.New("event: event not found")
	ErrListEvents       = errors.New("event: error listing events")
)
