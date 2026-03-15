package event

import (
	"encoding/json"
)

type CreateEventRequest struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}
