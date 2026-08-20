package types

import (
	"time"

	"github.com/google/uuid"
)

type Project struct {
	ID          uuid.UUID  `db:"id" json:"id" description:"The project's ID"`
	BusinessID  uuid.UUID  `db:"business_id" json:"business_id" description:"The business this project belongs to"`
	Title       string     `db:"title" json:"title" description:"The project's title"`
	Description *string    `db:"description" json:"description" description:"The project's description"`
	Image       *string    `db:"image" json:"image" description:"Showcase image URL"`
	URL         *string    `db:"url" json:"url" description:"An optional link to the project or a case study"`
	CompletedAt *time.Time `db:"completed_at" json:"completed_at" description:"When the project was completed, if known"`
	SubmittedBy uuid.UUID  `db:"submitted_by" json:"submitted_by" description:"The Metro user ID of whoever posted it"`
	Reviewer    *uuid.UUID `db:"reviewer" json:"reviewer" description:"The staff member (Metro user ID) who claimed this project for review"`
	Status      State      `db:"status" json:"status" description:"The project's review status: pending, under_review, approved, denied or suspended"`
	AvgRating   float64    `db:"avg_rating" json:"avg_rating" description:"The project's average star rating"`
	ReviewCount int        `db:"review_count" json:"review_count" description:"The project's total review count"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at" description:"When the project was posted"`
	UpdatedAt   time.Time  `db:"updated_at" json:"updated_at" description:"When the project was last updated"`
}

type ProjectCreate struct {
	Title       string     `json:"title" validate:"required" msg:"A title is required" description:"The project's title"`
	Description *string    `json:"description" description:"The project's description"`
	Image       *string    `json:"image" description:"Showcase image URL"`
	URL         *string    `json:"url" description:"An optional link to the project or a case study"`
	CompletedAt *time.Time `json:"completed_at" description:"When the project was completed, if known"`
}

type ProjectUpdate struct {
	Title       *string    `json:"title" description:"New title"`
	Description *string    `json:"description" description:"New description"`
	Image       *string    `json:"image" description:"New showcase image URL"`
	URL         *string    `json:"url" description:"New link"`
	CompletedAt *time.Time `json:"completed_at" description:"New completion date"`
}
