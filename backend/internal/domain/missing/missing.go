package missing

import "time"

// Person is a missing-person report aggregated into the hub.
type Person struct {
	ID             string     `json:"id"`
	Source         string     `json:"source"`
	ExternalID     string     `json:"external_id"`
	FullName       string     `json:"full_name"`
	Age            *int       `json:"age,omitempty"`
	Description    string     `json:"description"`
	LastSeenRegion string     `json:"last_seen_region"`
	Lat            *float64   `json:"lat,omitempty"`
	Lng            *float64   `json:"lng,omitempty"`
	LastSeenAt     *time.Time `json:"last_seen_at,omitempty"`
	Status         string     `json:"status"`
	Contact        string     `json:"contact"`
	PhotoURL       string     `json:"photo_url"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}
