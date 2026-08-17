package types

import (
	"time"

	"github.com/google/uuid"
)

type Action int

const (
	ActionClaim Action = iota
	ActionUnclaim
	ActionApprove
	ActionDeny
)

type State int

const (
	StatePending State = iota
	StateUnderReview
	StateApproved
	StateDenied
)

type ListState int

const (
	ListStatePendingAPISupport ListState = iota
	ListStateSupported
	ListStateDefunct
	ListStateBlacklisted
	ListStateUnconfirmedEnrollment
)

var GoodListStates = []ListState{ListStatePendingAPISupport, ListStateSupported}

func IsGoodListState(s ListState) bool {
	for _, g := range GoodListStates {
		if g == s {
			return true
		}
	}
	return false
}

type List struct {
	ID          uuid.UUID `db:"id" json:"id" description:"The unique ID of the list"`
	Name        string    `db:"name" json:"name" description:"The name of the list"`
	Description *string   `db:"description" json:"description" description:"The description of the list"`
	Domain      *string   `db:"domain" json:"domain" description:"The domain of the list"`
	State       ListState `db:"state" json:"state" description:"The current state of the list"`
	Icon        *string   `db:"icon" json:"icon" description:"The icon URL of the list"`
}

type ListUpdate struct {
	Name           *string `json:"name" description:"New name for the list"`
	Description    *string `json:"description" description:"New description for the list"`
	Domain         *string `json:"domain" description:"New domain for the list"`
	ClaimBotAPI    *string `json:"claim_bot_api" description:"Webhook called on claim"`
	UnclaimBotAPI  *string `json:"unclaim_bot_api" description:"Webhook called on unclaim"`
	ApproveBotAPI  *string `json:"approve_bot_api" description:"Webhook called on approve"`
	DenyBotAPI     *string `json:"deny_bot_api" description:"Webhook called on deny"`
	ResetSecretKey bool    `json:"reset_secret_key" description:"If true (and no other field is set), rotates the list secret key"`
	Icon           *string `json:"icon" description:"New icon URL (must be https)"`
}

type BotPost struct {
	BotID           string   `json:"bot_id" validate:"required" msg:"Bot ID is required" description:"The bot's Discord ID"`
	Banner          *string  `json:"banner" description:"Banner image URL"`
	Description     string   `json:"description" validate:"required" msg:"Description is required" description:"Short description"`
	LongDescription string   `json:"long_description" validate:"required" msg:"Long description is required" description:"Long description"`
	Website         *string  `json:"website" description:"Website URL"`
	Invite          *string  `json:"invite" description:"Bot invite URL"`
	Owner           string   `json:"owner" validate:"required" msg:"Owner is required" description:"The main owner's Discord ID"`
	ExtraOwners     []string `json:"extra_owners" description:"Additional owner Discord IDs"`
	Support         *string  `json:"support" description:"Support server URL"`
	Donate          *string  `json:"donate" description:"Donation link"`
	Library         *string  `json:"library" description:"Library the bot is written in"`
	NSFW            bool     `json:"nsfw" description:"Whether the bot is NSFW"`
	Prefix          *string  `json:"prefix" description:"Bot command prefix"`
	Tags            []string `json:"tags" description:"Bot tags"`
	ReviewNote      *string  `json:"review_note" description:"Note shown to reviewers"`
	CrossAdd        *bool    `json:"cross_add" description:"Whether the bot may be cross-added to other lists"`
}

type Bot struct {
	BotID           int64     `db:"bot_id" json:"bot_id" description:"The bot's Discord ID"`
	Username        string    `db:"username" json:"username" description:"The bot's username"`
	Banner          *string   `db:"banner" json:"banner" description:"Banner image URL"`
	Description     string    `db:"description" json:"description" description:"Short description"`
	LongDescription string    `db:"long_description" json:"long_description" description:"Long description"`
	Website         *string   `db:"website" json:"website" description:"Website URL"`
	Invite          *string   `db:"invite" json:"invite" description:"Bot invite URL"`
	Owner           int64     `db:"owner" json:"owner" description:"The main owner's Discord ID"`
	ExtraOwners     []int64   `db:"extra_owners" json:"extra_owners" description:"Additional owner Discord IDs"`
	Support         *string   `db:"support" json:"support" description:"Support server URL"`
	Donate          *string   `db:"donate" json:"donate" description:"Donation link"`
	Library         *string   `db:"library" json:"library" description:"Library the bot is written in"`
	NSFW            bool      `db:"nsfw" json:"nsfw" description:"Whether the bot is NSFW"`
	Prefix          *string   `db:"prefix" json:"prefix" description:"Bot command prefix"`
	Tags            []string  `db:"tags" json:"tags" description:"Bot tags"`
	ReviewNote      *string   `db:"review_note" json:"review_note" description:"Note shown to reviewers"`
	CrossAdd        bool      `db:"cross_add" json:"cross_add" description:"Whether the bot may be cross-added"`
	State           State     `db:"state" json:"state" description:"Current review state"`
	ListSource      uuid.UUID `db:"list_source" json:"list_source" description:"The list that submitted the bot"`
	AddedAt         time.Time `db:"added_at" json:"added_at" description:"When the bot was added"`
	Reviewer        *int64    `db:"reviewer" json:"reviewer" description:"The reviewer's Discord ID"`
	InviteLink      *string   `db:"invite_link" json:"invite_link" description:"Resolved invite link"`
}

type ActionRow struct {
	ID         uuid.UUID `db:"id" json:"id" description:"The action ID"`
	BotID      int64     `db:"bot_id" json:"bot_id" description:"The bot's Discord ID"`
	Action     Action    `db:"action" json:"action" description:"The action type"`
	Reason     string    `db:"reason" json:"reason" description:"Reason given for the action"`
	Reviewer   string    `db:"reviewer" json:"reviewer" description:"The reviewer's Discord ID"`
	ActionTime time.Time `db:"action_time" json:"action_time" description:"When the action was taken"`
	ListSource uuid.UUID `db:"list_source" json:"list_source" description:"The relevant list (not always present)"`
}

type Reason struct {
	Reason string `json:"reason" validate:"required" msg:"A reason is required" description:"The reason for the action"`
}

type TeamMember struct {
	Username    string   `json:"username" description:"The member's username"`
	ID          string   `json:"id" description:"The member's Discord ID"`
	Avatar      string   `json:"avatar" description:"The member's avatar URL"`
	IsListOwner bool     `json:"is_list_owner" description:"Whether the member holds the List Owner role"`
	Sudo        bool     `json:"sudo" description:"Whether the member holds the Sudo role"`
	Roles       []string `json:"roles" description:"List-related role names the member holds"`
}

type ApiError struct {
	Message string            `json:"message" description:"A human readable message describing the result"`
	Error   bool              `json:"error" description:"Whether this response represents an error"`
	Context map[string]string `json:"context,omitempty" description:"Additional context, e.g. per-field validation errors"`
}

type UpdatedResponse struct {
	HasUpdated []string `json:"has_updated" description:"The fields that were updated"`
}

type SecretKeyResponse struct {
	SecretKey string `json:"secret_key" description:"The new secret key"`
}

type PostBotResponse struct {
	Removed []string `json:"removed" description:"Fields that were dropped/sanitized from the submission"`
}
