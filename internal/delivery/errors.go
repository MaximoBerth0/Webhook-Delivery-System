package delivery

import "errors"

var (
	ErrInvalidEventID    = errors.New("delivery: event id is required")
	ErrInvalidWebhookID  = errors.New("delivery: webhook id is required")
	ErrInvalidDeliveryID = errors.New("delivery: delivery id is required")
	ErrCreateDelivery    = errors.New("delivery: error creating delivery")
	ErrDeliveryNotFound  = errors.New("delivery: delivery not found")
	ErrGetPending        = errors.New("delivery: error getting pending deliveries")
	ErrGetRetryable      = errors.New("delivery: error getting retryable deliveries")
	ErrUpdateStatus      = errors.New("delivery: error updating delivery status")
	ErrIncrementAttempts = errors.New("delivery: error incrementing delivery attempts")
)
