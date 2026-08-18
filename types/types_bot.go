package types

import (
	"time"

	"github.com/google/uuid"
)

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
}

type Bot struct {
	BotID           int64      `db:"bot_id" json:"bot_id" description:"The bot's Discord ID"`
	Username        string     `db:"username" json:"username" description:"The bot's username"`
	Banner          *string    `db:"banner" json:"banner" description:"Banner image URL"`
	Description     string     `db:"description" json:"description" description:"Short description"`
	LongDescription string     `db:"long_description" json:"long_description" description:"Long description"`
	Website         *string    `db:"website" json:"website" description:"Website URL"`
	Invite          *string    `db:"invite" json:"invite" description:"Bot invite URL"`
	Owner           int64      `db:"owner" json:"owner" description:"The main owner's Discord ID"`
	ExtraOwners     []int64    `db:"extra_owners" json:"extra_owners" description:"Additional owner Discord IDs"`
	Support         *string    `db:"support" json:"support" description:"Support server URL"`
	Donate          *string    `db:"donate" json:"donate" description:"Donation link"`
	Library         *string    `db:"library" json:"library" description:"Library the bot is written in"`
	NSFW            bool       `db:"nsfw" json:"nsfw" description:"Whether the bot is NSFW"`
	Prefix          *string    `db:"prefix" json:"prefix" description:"Bot command prefix"`
	Tags            []string   `db:"tags" json:"tags" description:"Bot tags"`
	ReviewNote      *string    `db:"review_note" json:"review_note" description:"Note shown to reviewers"`
	State           State      `db:"state" json:"state" description:"Current review state"`
	AvgRating       float64    `db:"avg_rating" json:"avg_rating" description:"The bot's average star rating"`
	ReviewCount     int        `db:"review_count" json:"review_count" description:"The bot's total review count"`
	AddedAt         time.Time  `db:"added_at" json:"added_at" description:"When the bot was added"`
	Reviewer        *uuid.UUID `db:"reviewer" json:"reviewer" description:"The reviewer's Metro user ID"`
	InviteLink      *string    `db:"invite_link" json:"invite_link" description:"Resolved invite link"`
}

type PostBotResponse struct {
	Removed []string `json:"removed" description:"Fields that were dropped/sanitized from the submission"`
}
