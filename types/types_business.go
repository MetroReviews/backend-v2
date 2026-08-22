package types

import (
	"time"

	"github.com/google/uuid"
)

type Category struct {
	ID          uuid.UUID `db:"id" json:"id" description:"The category's ID"`
	Slug        string    `db:"slug" json:"slug" description:"The category's URL slug"`
	Name        string    `db:"name" json:"name" description:"The category's display name"`
	Description *string   `db:"description" json:"description" description:"The category's description"`
	Icon        *string   `db:"icon" json:"icon" description:"The category's icon URL"`
}

type Business struct {
	ID            uuid.UUID      `db:"id" json:"id" description:"The business's ID"`
	CategoryID    uuid.UUID      `db:"category_id" json:"category_id" description:"The business's category ID"`
	Slug          string         `db:"slug" json:"slug" description:"The business's URL slug"`
	Name          string         `db:"name" json:"name" description:"The business's name"`
	Description   *string        `db:"description" json:"description" description:"The business's description"`
	Website       *string        `db:"website" json:"website" description:"The business's website URL"`
	Logo          *string        `db:"logo" json:"logo" description:"The business's logo URL"`
	Banner        *string        `db:"banner" json:"banner" description:"The business's banner URL"`
	Address       *string        `db:"address" json:"address" description:"The business's street address"`
	City          *string        `db:"city" json:"city" description:"The business's city"`
	Country       *string        `db:"country" json:"country" description:"The business's country"`
	Metadata      map[string]any `db:"metadata" json:"metadata" description:"Category-specific extra fields"`
	OwnerID       *uuid.UUID     `db:"owner_id" json:"owner_id" description:"The verified owner's Metro user ID, if an ownership claim has been approved"`
	SubmittedBy   uuid.UUID      `db:"submitted_by" json:"submitted_by" description:"The Metro user ID of whoever submitted the business"`
	Status        State          `db:"status" json:"status" description:"The business's review status: pending, under_review, approved, denied or suspended"`
	Reviewer      *uuid.UUID     `db:"reviewer" json:"reviewer" description:"The staff member (Metro user ID) who claimed this business for review"`
	AvgRating     float64        `db:"avg_rating" json:"avg_rating" description:"The business's average star rating"`
	ReviewCount   int            `db:"review_count" json:"review_count" description:"The business's total review count"`
	CreatedAt     time.Time      `db:"created_at" json:"created_at" description:"When the business was created"`
	UpdatedAt     time.Time      `db:"updated_at" json:"updated_at" description:"When the business was last updated"`
	Latitude      *float64       `db:"latitude" json:"latitude" description:"Latitude, for \"near me\" search"`
	Longitude     *float64       `db:"longitude" json:"longitude" description:"Longitude, for \"near me\" search"`
	Gallery       []string       `db:"gallery" json:"gallery" description:"Showcase photo URLs, separate from logo/banner"`
	Featured      bool           `db:"featured" json:"featured" description:"Whether this business has sponsored/featured placement"`
	FeaturedUntil *time.Time     `db:"featured_until" json:"featured_until" description:"When the featured placement expires, if featured"`
	ViewCount     int            `db:"view_count" json:"view_count" description:"Total profile views"`
}

type BusinessSearchResult struct {
	Business
	DistanceKM *float64 `db:"distance_km" json:"distance_km,omitempty" description:"Distance in km from the given lat/lng, if searched by location"`
}

type BusinessCreate struct {
	CategoryID  uuid.UUID      `json:"category_id" validate:"required" msg:"A category is required" description:"The business's category ID"`
	Slug        string         `json:"slug" description:"Optional URL slug; generated from the name if omitted"`
	Name        string         `json:"name" validate:"required" msg:"A name is required" description:"The business's name"`
	Description *string        `json:"description" description:"The business's description"`
	Website     *string        `json:"website" description:"The business's website URL"`
	Logo        *string        `json:"logo" description:"The business's logo URL"`
	Banner      *string        `json:"banner" description:"The business's banner URL"`
	Address     *string        `json:"address" description:"The business's street address"`
	City        *string        `json:"city" description:"The business's city"`
	Country     *string        `json:"country" description:"The business's country"`
	Metadata    map[string]any `json:"metadata" description:"Category-specific extra fields"`
	Latitude    *float64       `json:"latitude" description:"Latitude, for \"near me\" search"`
	Longitude   *float64       `json:"longitude" description:"Longitude, for \"near me\" search"`
	Gallery     []string       `json:"gallery" description:"Showcase photo URLs (https only, max 12)"`
}

type BusinessUpdate struct {
	Name        *string        `json:"name" description:"New name for the business"`
	Description *string        `json:"description" description:"New description for the business"`
	Website     *string        `json:"website" description:"New website URL"`
	Logo        *string        `json:"logo" description:"New logo URL"`
	Banner      *string        `json:"banner" description:"New banner URL"`
	Address     *string        `json:"address" description:"New street address"`
	City        *string        `json:"city" description:"New city"`
	Country     *string        `json:"country" description:"New country"`
	Metadata    map[string]any `json:"metadata" description:"New category-specific extra fields"`
	Latitude    *float64       `json:"latitude" description:"New latitude"`
	Longitude   *float64       `json:"longitude" description:"New longitude"`
	Gallery     []string       `json:"gallery" description:"New showcase photo URLs (https only, max 12), replaces the whole list"`
}

type BusinessFeature struct {
	Featured bool       `json:"featured" description:"Whether the business should be featured"`
	Until    *time.Time `json:"until" description:"When the featuring expires; omit/null for indefinite"`
}

type ClaimCreate struct {
	Note *string `json:"note" description:"An optional note supporting the ownership claim"`
}

type Claim struct {
	ID         uuid.UUID   `db:"id" json:"id" description:"The claim's ID"`
	BusinessID uuid.UUID   `db:"business_id" json:"business_id" description:"The business being claimed"`
	UserID     uuid.UUID   `db:"user_id" json:"user_id" description:"The Metro user ID of whoever filed the claim"`
	Note       *string     `db:"note" json:"note" description:"An optional note supporting the claim"`
	Status     ClaimStatus `db:"status" json:"status" description:"The claim's current status"`
	CreatedAt  time.Time   `db:"created_at" json:"created_at" description:"When the claim was filed"`
	ResolvedBy *uuid.UUID  `db:"resolved_by" json:"resolved_by" description:"The staff member (Metro user ID) who resolved the claim"`
	ResolvedAt *time.Time  `db:"resolved_at" json:"resolved_at" description:"When the claim was resolved"`
}
