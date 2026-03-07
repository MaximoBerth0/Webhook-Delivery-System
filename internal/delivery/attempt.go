package delivery

import "time"

type Attempt struct {
	ID         string
	DeliveryID string

	AttemptNumber int
	Status        Status

	ResponseCode int
	ErrorMessage string

	CreatedAt time.Time
}
