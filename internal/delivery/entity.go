package delivery

type Status string

const (
	StatusPending Status = "PENDING"
	StatusSuccess Status = "SUCCESS"
	StatusFailed  Status = "FAILED"
	StatusRetry   Status = "RETRY"
)

type Delivery struct {
	ID        string
	EventID   string
	WebhookID string
	Attempts  int
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
		Attempts:  0,
		Status:    StatusPending,
	}, nil
}

func (d *Delivery) RegisterAttempt() {
	d.Attempts++
}

func (d *Delivery) MarkSuccess() {
	d.Status = StatusSuccess
}

func (d *Delivery) MarkRetry() {
	d.Status = StatusRetry
}

func (d *Delivery) MarkFailed() {
	d.Status = StatusFailed
}
