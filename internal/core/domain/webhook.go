package domain

import "time"

type WebhookConfig struct {
    URL        string           `json:"url"`
    Secret     string           `json:"secret"`
    EventTypes []WebhookEvent   `json:"event_types"`
}

type WebhookEvent string

const (
    EventJobCompleted WebhookEvent = "job.completed"
    EventJobFailed    WebhookEvent = "job.failed"
    EventJobPaused    WebhookEvent = "job.paused"
    EventJobResumed   WebhookEvent = "job.resumed"
)

type WebhookPayload struct {
    EventType WebhookEvent             `json:"event_type"`
    Timestamp time.Time                `json:"timestamp"`
    JobID     string                   `json:"job_id"`
    Data      map[string]interface{}   `json:"data"`
	Signature string                   `json:"signature"`
}
