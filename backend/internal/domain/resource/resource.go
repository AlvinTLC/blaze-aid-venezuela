package resource

import "time"

// Resource is a physical supply offered or needed in the field.
type Resource struct {
	ID         string    `json:"id"`
	Source     string    `json:"source"`
	ExternalID string    `json:"external_id"`
	Type       string    `json:"type"`
	Name       string    `json:"name"`
	Quantity   float64   `json:"quantity"`
	Unit       string    `json:"unit"`
	Status     string    `json:"status"`
	Region     string    `json:"region"`
	Lat        *float64  `json:"lat,omitempty"`
	Lng        *float64  `json:"lng,omitempty"`
	Contact    string    `json:"contact"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
