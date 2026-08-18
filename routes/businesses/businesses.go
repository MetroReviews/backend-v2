// Package businesses exposes the /businesses endpoints: the generic
// Yelp/Trustpilot-style side of Metro — any reviewable service or business,
// browsable by category, claimable by its owner. Handlers are split one
// concern per file; route registration is split the same way, core CRUD
// here and review-queue/ownership-claim registration in
// businesses_review_routes.go — this file just wires the two together.
package businesses

import (
	"github.com/go-chi/chi/v5"
)

const tagName = "Businesses"

const businessColumns = `id, category_id, slug, name, description, website, logo, banner, address, city, country, metadata, owner_id, submitted_by, status, avg_rating, review_count, created_at, updated_at`

type Router struct{}

func (Router) Tag() (string, string) {
	return tagName, "Reviewable businesses: any service or business on Metro."
}

func (Router) Routes(r *chi.Mux) {
	registerCoreRoutes(r)
	registerReviewRoutes(r)
}
