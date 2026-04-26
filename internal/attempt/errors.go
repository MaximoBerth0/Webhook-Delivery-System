package attempt

import "errors"

var (
	ErrInvalidDeliveryID    = errors.New("attempt: delivery id is required")
	ErrInvalidAttemptNumber = errors.New("attempt: attempt number must be greater than 0")
)
