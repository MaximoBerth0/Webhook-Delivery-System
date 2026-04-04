package delivery

import (
	"errors"
)

var ErrInvalidDeliveryID = errors.New("delivery id not found")

var ErrCreateDelivery = errors.New("error creating delivery")

var ErrDeliveryNotFound = errors.New("error delivery not found")

var ErrGetPending = errors.New("error getting pending deliveries")

var ErrGetRetryable = errors.New("error getting retryable deliveries")

var ErrUpdateStatus = errors.New("error updating delivery status")

var ErrIncrementAttempts = errors.New("error incrementing delivery attempts")
