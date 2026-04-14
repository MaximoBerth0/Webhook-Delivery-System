package attempt

import (
	"errors"
	"time"
)

var ErrInvalidDeliveryID = errors.New("invalid delivery id")

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

func NewDeliveryAttempt(deliveryID string) (*DeliveryAttempt, error) {
	if deliveryID == "" {
		return nil, ErrInvalidDeliveryID
	}
	return &DeliveryAttempt{
		DeliveryID: deliveryID,
		Status:     StatusPending,
		CreatedAt:  time.Now(),
	}, nil
}
