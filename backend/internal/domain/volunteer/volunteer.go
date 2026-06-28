package volunteer

import "time"

// Volunteer is a person offering skills or time to relief efforts.
type Volunteer struct {
	ID           string    `json:"id"`
	Source       string    `json:"source"`
	ExternalID   string    `json:"external_id"`
	FullName     string    `json:"full_name"`
	Skills       []string  `json:"skills"`
	Availability string    `json:"availability"`
	Region       string    `json:"region"`
	Contact      string    `json:"contact"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
