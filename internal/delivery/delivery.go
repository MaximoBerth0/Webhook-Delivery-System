package delivery

import "errors"

type Status string

const (
	StatusPending Status = "PENDING"
	StatusSuccess Status = "SUCCESS"
	StatusFailed  Status = "FAILED"
	StatusRetry   Status = "RETRY"
)

var ErrInvalidAttempts = errors.New("max attempts must be >= 1")

type Delivery struct {
	ID          string
	EventID     string
	WebhookID   string
	Attempts    int
	MaxAttempts int
	Status      Status
}

func NewDelivery(id, eventID, webhookID string, maxAttempts int) (*Delivery, error) {
	if maxAttempts < 1 {
		return nil, ErrInvalidAttempts
	}

	return &Delivery{
		ID:          id,
		EventID:     eventID,
		WebhookID:   webhookID,
		Attempts:    0,
		MaxAttempts: maxAttempts,
		Status:      StatusPending,
	}, nil
}

func (d *Delivery) RegisterAttempt() {
	d.Attempts++
}

func (d *Delivery) CanRetry() bool {
	return d.Attempts < d.MaxAttempts
}

func (d *Delivery) MarkSuccess() {
	d.Status = StatusSuccess
}

func (d *Delivery) MarkFailed() {
	if d.CanRetry() {
		d.Status = StatusRetry
		return
	}

	d.Status = StatusFailed
}
