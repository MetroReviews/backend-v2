// Package reviews exposes the /reviews endpoints: the review engine shared
// by businesses, bots and projects. A review always belongs to exactly one
// of those three (see the reviews table's CHECK constraint); everything
// else (voting, owner replies, reporting) works the same regardless of
// which. Handlers are split one concern per file; route registration
// mirrors that split — core review CRUD/fetching here, voting/owner-
// response/reporting registration in routes_moderation.go — this file
// just wires the two together.
package reviews

import (
	"github.com/go-chi/chi/v5"
)

const tagName = "Reviews"

const reviewColumns = `id, business_id, bot_id, project_id, author_id, rating, title, body, owner_response, owner_response_at, helpful_count, status, created_at, updated_at`

type Router struct{}

func (Router) Tag() (string, string) {
	return tagName, "Reviews and ratings, shared by businesses, bots and projects."
}

func (Router) Routes(r *chi.Mux) {
	registerCoreRoutes(r)
	registerModerationRoutes(r)
}
