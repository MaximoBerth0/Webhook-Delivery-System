package delivery

import "time"

type DeliveryAttempt struct {
	ID         string
	DeliveryID string

	AttemptNumber int
	Status        Status

	ResponseCode int
	ErrorMessage string

	CreatedAt time.Time
}
