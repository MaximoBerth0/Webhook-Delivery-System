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
	ExecutedAt    *time.Time
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

func (d *DeliveryAttempt) MarkAsSuccess(statusCode int) error {
	if d.Status != StatusPending {
		return ErrAttemptNotPending
	}

	d.Status = StatusSuccess
	d.ResponseCode = statusCode
	now := time.Now()
	d.ExecutedAt = &now

	return nil
}

func (d *DeliveryAttempt) MarkAsFailed(statusCode int, errorMessage string) error {
	if d.Status != StatusPending {
		return ErrAttemptNotPending
	}

	d.Status = StatusFailed
	d.ResponseCode = statusCode
	d.ErrorMessage = errorMessage
	now := time.Now()
	d.ExecutedAt = &now

	return nil
}

func (d *DeliveryAttempt) IsReadyToExecute() bool {
	return d.Status == StatusPending && time.Now().After(d.ScheduledFor)
}
