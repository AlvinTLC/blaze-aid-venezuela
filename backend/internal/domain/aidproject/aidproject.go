package aidproject

import "time"

// AidProject is a tech/relief project aggregated into the BlazeAid Hub.
type AidProject struct {
	ID          string    `json:"id"`
	Source      string    `json:"source"`
	ExternalID  string    `json:"external_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	Status      string    `json:"status"`
	Region      string    `json:"region"`
	Lat         *float64  `json:"lat,omitempty"`
	Lng         *float64  `json:"lng,omitempty"`
	Contact     string    `json:"contact"`
	URL         string    `json:"url"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
