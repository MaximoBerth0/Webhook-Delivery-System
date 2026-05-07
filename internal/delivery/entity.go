package delivery

type Status string

const (
	StatusPending Status = "PENDING"
	StatusSuccess Status = "SUCCESS"
	StatusFailed  Status = "FAILED"
)

type Delivery struct {
	ID        string
	EventID   string
	WebhookID string
	Status    Status
}

func NewDelivery(eventID, webhookID string) (*Delivery, error) {
	if eventID == "" {
		return nil, ErrInvalidEventID
	}
	if webhookID == "" {
		return nil, ErrInvalidWebhookID
	}

	return &Delivery{
		EventID:   eventID,
		WebhookID: webhookID,
		Status:    StatusPending,
	}, nil
}

func (d *Delivery) MarkSuccess() {
	d.Status = StatusSuccess
}

func (d *Delivery) MarkFailed() {
	d.Status = StatusFailed
}
