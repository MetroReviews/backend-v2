package types

import "github.com/google/uuid"

type SearchResult struct {
	Type        string    `db:"type" json:"type" description:"\"business\" or \"project\""`
	ID          uuid.UUID `db:"id" json:"id" description:"The business's or project's ID"`
	Slug        *string   `db:"slug" json:"slug,omitempty" description:"The business's URL slug (businesses only)"`
	Name        string    `db:"name" json:"name" description:"The business's name, or the project's title"`
	Description *string   `db:"description" json:"description"`
	AvgRating   float64   `db:"avg_rating" json:"avg_rating"`
	ReviewCount int       `db:"review_count" json:"review_count"`
	Rank        float64   `db:"rank" json:"rank" description:"Full-text search rank; higher is more relevant"`
}
