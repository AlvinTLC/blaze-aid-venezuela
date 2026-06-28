package jobs

import (
	"context"
	"encoding/json"

	"github.com/riverqueue/river"

	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/domain/aidproject"
	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/domain/missing"
	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/domain/resource"
	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/domain/volunteer"
	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/repository"
)

// WebhookProcessWorker consumes WebhookProcessArgs: it loads the raw payload,
// routes it to the right typed entity by the payload's `entity` field, records
// an event, and marks the webhook processed. Unknown/unparseable payloads are
// recorded as "unrouted" (not an error). Only real DB failures return an error,
// which lets River retry with its default backoff.
type WebhookProcessWorker struct {
	river.WorkerDefaults[WebhookProcessArgs]
	Repo *repository.Repository
}

// webhookPayload is a superset of the fields any single source might send.
type webhookPayload struct {
	Entity     string `json:"entity"`
	Source     string `json:"source"`
	ExternalID string `json:"external_id"`

	// project / shared
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Category    string  `json:"category"`
	Status      string  `json:"status"`
	Region      string  `json:"region"`
	URL         string  `json:"url"`
	Contact     string  `json:"contact"`
	Lat         *float64 `json:"lat"`
	Lng         *float64 `json:"lng"`

	// resource
	Type     string  `json:"type"`
	Name     string  `json:"name"`
	Quantity float64 `json:"quantity"`
	Unit     string  `json:"unit"`

	// missing
	FullName       string `json:"full_name"`
	Age            *int   `json:"age"`
	LastSeenRegion string `json:"last_seen_region"`
	PhotoURL       string `json:"photo_url"`

	// volunteer
	Skills       []string `json:"skills"`
	Availability string   `json:"availability"`
}

func (w *WebhookProcessWorker) Work(ctx context.Context, job *river.Job[WebhookProcessArgs]) error {
	log, err := w.Repo.GetWebhookLog(ctx, job.Args.WebhookID)
	if err != nil {
		return err
	}
	if log.Processed { // idempotent: already handled on a prior attempt
		return nil
	}

	entity, entityID, err := w.route(ctx, job.Args, log)
	if err != nil {
		return err // real DB error → let River retry
	}

	eventEntity := entity
	kind := entity
	if entity == "" {
		eventEntity, kind = "webhook", "unrouted"
	}
	if err := w.Repo.RecordEvent(ctx, eventEntity, entityID, kind, log.Payload); err != nil {
		return err
	}
	return w.Repo.MarkWebhookProcessed(ctx, job.Args.WebhookID, "processed")
}

// route upserts the payload into its typed table. Returns ("","",nil) when the
// payload can't be routed (unknown/unparseable) — a non-error outcome.
func (w *WebhookProcessWorker) route(ctx context.Context, args WebhookProcessArgs, log repository.WebhookLog) (entity, entityID string, err error) {
	var p webhookPayload
	if json.Unmarshal(log.Payload, &p) != nil {
		return "", "", nil
	}

	source := firstNonEmpty(p.Source, args.Source, log.Source)
	externalID := firstNonEmpty(p.ExternalID, args.WebhookID)

	switch p.Entity {
	case "project":
		id, err := w.Repo.UpsertProject(ctx, aidproject.AidProject{
			Source: source, ExternalID: externalID, Title: p.Title, Description: p.Description,
			Category: p.Category, Status: p.Status, Region: p.Region, Lat: p.Lat, Lng: p.Lng,
			Contact: p.Contact, URL: p.URL,
		})
		return "project", id, err
	case "resource":
		id, err := w.Repo.UpsertResource(ctx, resource.Resource{
			Source: source, ExternalID: externalID, Type: p.Type, Name: p.Name,
			Quantity: p.Quantity, Unit: p.Unit, Status: p.Status, Region: p.Region,
			Lat: p.Lat, Lng: p.Lng, Contact: p.Contact,
		})
		return "resource", id, err
	case "missing":
		id, err := w.Repo.UpsertMissing(ctx, missing.Person{
			Source: source, ExternalID: externalID, FullName: p.FullName, Age: p.Age,
			Description: p.Description, LastSeenRegion: p.LastSeenRegion, Status: p.Status,
			Contact: p.Contact, PhotoURL: p.PhotoURL,
		})
		return "missing", id, err
	case "volunteer":
		skills := p.Skills
		if skills == nil {
			skills = []string{}
		}
		id, err := w.Repo.UpsertVolunteer(ctx, volunteer.Volunteer{
			Source: source, ExternalID: externalID, FullName: p.FullName, Skills: skills,
			Availability: p.Availability, Region: p.Region, Contact: p.Contact, Status: p.Status,
		})
		return "volunteer", id, err
	default:
		return "", "", nil
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
