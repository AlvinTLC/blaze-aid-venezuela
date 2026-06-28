package jobs

// WebhookProcessArgs is the River job enqueued for each inbound webhook. The
// payload itself lives in webhooks_log; the job only carries the reference.
type WebhookProcessArgs struct {
	WebhookID string `json:"webhook_id"`
	Source    string `json:"source"`
}

// Kind is the stable River job type identifier.
func (WebhookProcessArgs) Kind() string { return "webhook_process" }
