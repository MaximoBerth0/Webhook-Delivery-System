package attempt

import (
	"time"
)

type Status string

const (
	StatusPending Status = "pending"
	StatusSuccess Status = "success"
	StatusFailed  Status = "failed"
)

type DeliveryAttempt struct {
	ID            string
	DeliveryID    string
	AttemptNumber int
	Status        Status
	ScheduledFor  time.Time
	ResponseCode  int
	ErrorMessage  string
	CreatedAt     time.Time
}

func NewDeliveryAttempt(deliveryID string, attemptNumber int, scheduledFor time.Time) (*DeliveryAttempt, error) {
	if deliveryID == "" {
		return nil, ErrInvalidDeliveryID
	}
	if attemptNumber <= 0 {
		return nil, ErrInvalidAttemptNumber
	}

	return &DeliveryAttempt{
		DeliveryID:    deliveryID,
		Status:        StatusPending,
		AttemptNumber: attemptNumber,
		ScheduledFor:  scheduledFor,
		CreatedAt:     time.Now(),
	}, nil
}
