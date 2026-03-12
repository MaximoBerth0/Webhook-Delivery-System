package attempt

import "time"

type Status string

const (
	StatusPending   Status = "pending"
	StatusSuccess   Status = "success"
	StatusFailed    Status = "failed"
	StatusCancelled Status = "cancelled"
)

type DeliveryAttempt struct {
	ID            string
	DeliveryID    string
	AttemptNumber int
	Status        Status
	ResponseCode  int
	ErrorMessage  string
	CreatedAt     time.Time
}
