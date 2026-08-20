package types

import (
	"time"

	"github.com/google/uuid"
)

type Webhook struct {
	ID              uuid.UUID  `db:"id" json:"id" description:"The webhook's ID"`
	TargetType      string     `db:"target_type" json:"target_type" description:"What this webhook is registered against: business, project, ..."`
	TargetID        string     `db:"target_id" json:"target_id" description:"The ID of the target (a business/project UUID)"`
	URL             string     `db:"url" json:"url" description:"Where events are POSTed"`
	Secret          string     `db:"secret" json:"-" description:"HMAC-SHA256 signing secret — never serialized; see WebhookRevealed"`
	Events          []string   `db:"events" json:"events" description:"Event names to deliver; empty means every event"`
	CreatedBy       uuid.UUID  `db:"created_by" json:"created_by" description:"The Metro user ID who registered this webhook"`
	Enabled         bool       `db:"enabled" json:"enabled" description:"Whether deliveries are currently attempted"`
	FailureCount    int        `db:"failure_count" json:"failure_count" description:"Consecutive delivery failures; auto-disabled past a threshold"`
	LastTriggeredAt *time.Time `db:"last_triggered_at" json:"last_triggered_at" description:"When a delivery last succeeded"`
	CreatedAt       time.Time  `db:"created_at" json:"created_at" description:"When the webhook was registered"`
	UpdatedAt       time.Time  `db:"updated_at" json:"updated_at" description:"When the webhook was last updated"`
}

type WebhookRevealed struct {
	Webhook
	Secret string `json:"secret" description:"The webhook's signing secret — shown once, save it now. Sign deliveries by HMAC-SHA256'ing the raw request body with it and comparing against the X-Metro-Signature header."`
}

type WebhookCreate struct {
	TargetType string   `json:"target_type" validate:"required" msg:"A target_type is required" description:"What to register this webhook against: business, project, ..."`
	TargetID   string   `json:"target_id" validate:"required" msg:"A target_id is required" description:"The ID of the target (a business/project UUID)"`
	URL        string   `json:"url" validate:"required,httpurl" msg:"A valid http(s) URL is required" description:"Where events are POSTed"`
	Events     []string `json:"events" description:"Event names to subscribe to; omit/empty for every event"`
}

type WebhookUpdate struct {
	URL     *string  `json:"url" description:"New delivery URL"`
	Events  []string `json:"events" description:"New event subscription list, replacing the existing one; empty means every event"`
	Enabled *bool    `json:"enabled" description:"Pause (false) or resume (true) deliveries"`
}

type WebhookEvent struct {
	Name        string `json:"name" description:"The event name, as it appears in Webhook.Events and the delivered payload's \"event\" field"`
	Description string `json:"description" description:"What triggers this event"`
}
