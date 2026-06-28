package sync

import (
	"encoding/json"
	"time"
)

// Change is a single entity mutation surfaced to clients via GET /sync.
// Data carries the full entity row as JSON so clients stay schema-agnostic.
type Change struct {
	Entity    string          `json:"entity"`
	ID        string          `json:"id"`
	UpdatedAt time.Time       `json:"updated_at"`
	Data      json.RawMessage `json:"data"`
}
