package types

import (
	"time"

	"github.com/google/uuid"
)

type ReviewCreate struct {
	BusinessID  *uuid.UUID `json:"business_id" description:"The business being reviewed (mutually exclusive with project_id)"`
	ProjectID   *uuid.UUID `json:"project_id" description:"The project being reviewed (mutually exclusive with business_id)"`
	Rating      int16      `json:"rating" validate:"required,min=1,max=5" msg:"A rating from 1 to 5 is required" description:"Star rating, 1 to 5"`
	Title       *string    `json:"title" description:"An optional review title"`
	Body        string     `json:"body" validate:"required" msg:"Review text is required" description:"The review's text"`
	Photos      []string   `json:"photos" description:"Photo URLs (https only, max 6)"`
	InviteToken *string    `json:"invite_token" description:"A review-invitation token (see POST /businesses/{id}/invites) — redeeming a valid one marks this review verified"`
}

type ReviewUpdate struct {
	Rating *int16  `json:"rating" description:"New star rating, 1 to 5"`
	Title  *string `json:"title" description:"New review title"`
	Body   *string `json:"body" description:"New review text"`
}

type ReviewResponse struct {
	Response string `json:"response" validate:"required" msg:"A response is required" description:"The owner's reply to the review"`
}

type ReviewVote struct {
	Helpful bool `json:"helpful" description:"true for helpful, false for unhelpful"`
}

type Review struct {
	ID              uuid.UUID    `db:"id" json:"id" description:"The review's ID"`
	BusinessID      *uuid.UUID   `db:"business_id" json:"business_id" description:"The business being reviewed, if any"`
	ProjectID       *uuid.UUID   `db:"project_id" json:"project_id" description:"The project being reviewed, if any"`
	AuthorID        uuid.UUID    `db:"author_id" json:"author_id" description:"The Metro user ID of the reviewer"`
	Rating          int16        `db:"rating" json:"rating" description:"Star rating, 1 to 5"`
	Title           *string      `db:"title" json:"title" description:"The review's title"`
	Body            string       `db:"body" json:"body" description:"The review's text"`
	OwnerResponse   *string      `db:"owner_response" json:"owner_response" description:"The owner's reply, if any"`
	OwnerResponseAt *time.Time   `db:"owner_response_at" json:"owner_response_at" description:"When the owner replied"`
	HelpfulCount    int          `db:"helpful_count" json:"helpful_count" description:"Net helpful votes"`
	Status          ReviewStatus `db:"status" json:"status" description:"The review's current status"`
	CreatedAt       time.Time    `db:"created_at" json:"created_at" description:"When the review was posted"`
	UpdatedAt       time.Time    `db:"updated_at" json:"updated_at" description:"When the review was last edited"`
	Photos          []string     `db:"photos" json:"photos" description:"Photo URLs attached to the review"`
	FlagReason      *string      `db:"flag_reason" json:"flag_reason,omitempty" description:"Why the fraud package auto-flagged this review, if it did"`
	Verified        bool         `db:"verified" json:"verified" description:"Whether this review was posted by redeeming a review invitation"`
}

type ReviewInviteStatus int

const (
	InviteStatusPending ReviewInviteStatus = iota
	InviteStatusRedeemed
	InviteStatusExpired
)

type ReviewInviteCreate struct {
	TargetEmail string `json:"target_email" validate:"required,email" msg:"A valid email is required" description:"Who the invite is for"`
}

type ReviewInvite struct {
	ID               uuid.UUID          `db:"id" json:"id" description:"The invite's ID"`
	BusinessID       *uuid.UUID         `db:"business_id" json:"business_id" description:"The business to be reviewed, if any"`
	ProjectID        *uuid.UUID         `db:"project_id" json:"project_id" description:"The project to be reviewed, if any"`
	TargetEmail      string             `db:"target_email" json:"target_email" description:"Who the invite was sent to"`
	Token            string             `db:"token" json:"token" description:"The redemption token — pass as invite_token on POST /reviews"`
	CreatedBy        uuid.UUID          `db:"created_by" json:"created_by" description:"The Metro user ID who created the invite"`
	Status           ReviewInviteStatus `db:"status" json:"status" description:"pending, redeemed or expired"`
	RedeemedReviewID *uuid.UUID         `db:"redeemed_review_id" json:"redeemed_review_id" description:"The review it was redeemed into, if any"`
	ExpiresAt        time.Time          `db:"expires_at" json:"expires_at" description:"When the invite link stops working"`
	CreatedAt        time.Time          `db:"created_at" json:"created_at" description:"When the invite was created"`
}

type ReportCreate struct {
	Reason string `json:"reason" validate:"required" msg:"A reason is required" description:"Why this is being reported"`
}

type Report struct {
	ID         uuid.UUID    `db:"id" json:"id" description:"The report's ID"`
	TargetType string       `db:"target_type" json:"target_type" description:"What's being reported: review, business or project"`
	TargetID   string       `db:"target_id" json:"target_id" description:"The ID of the thing being reported"`
	ReporterID uuid.UUID    `db:"reporter_id" json:"reporter_id" description:"The Metro user ID of whoever filed the report"`
	Reason     string       `db:"reason" json:"reason" description:"Why this was reported"`
	Status     ReportStatus `db:"status" json:"status" description:"The report's current status"`
	CreatedAt  time.Time    `db:"created_at" json:"created_at" description:"When the report was filed"`
	ResolvedBy *uuid.UUID   `db:"resolved_by" json:"resolved_by" description:"The staff member (Metro user ID) who resolved the report"`
	ResolvedAt *time.Time   `db:"resolved_at" json:"resolved_at" description:"When the report was resolved"`
}
