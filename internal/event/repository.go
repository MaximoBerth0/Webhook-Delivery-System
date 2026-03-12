package event

import "context"

type Repository interface {
	Create(ctx context.Context, event *Event) error
	GetByID(ctx context.Context, id string) (*Event, error)
	List(ctx context.Context, limit int) ([]Event, error)
}
