package types

import (
	"time"

	"github.com/google/uuid"
)

// ModerationAction is one claim/unclaim/approve/deny record from the shared
// review queue, against either a bot or a business (TargetType/TargetID).
type ModerationAction struct {
	ID         uuid.UUID `db:"id" json:"id" description:"The action ID"`
	TargetType string    `db:"target_type" json:"target_type" description:"What was actioned: bot or business"`
	TargetID   string    `db:"target_id" json:"target_id" description:"The ID of the bot or business actioned"`
	Action     Action    `db:"action" json:"action" description:"The action type"`
	Reason     string    `db:"reason" json:"reason" description:"Reason given for the action"`
	Reviewer   uuid.UUID `db:"reviewer" json:"reviewer" description:"The reviewer's Metro user ID"`
	ActionTime time.Time `db:"action_time" json:"action_time" description:"When the action was taken"`
}

type Reason struct {
	Reason string `json:"reason" validate:"required" msg:"A reason is required" description:"The reason for the action"`
}
