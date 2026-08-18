// Package projects exposes the /projects endpoints: portfolio/showcase
// items a business posts, one business ID at a time — browsable off
// /businesses/{business_id}/projects, individually addressable at
// /projects/{id}. Handlers are split one concern per file; route
// registration is split the same way, core CRUD here and review-queue
// registration in routes_review.go — this file just wires the two together.
package projects

import (
	"github.com/go-chi/chi/v5"
)

const tagName = "Projects"

const projectColumns = `id, business_id, title, description, image, url, completed_at, submitted_by, reviewer, status, avg_rating, review_count, created_at, updated_at`

type Router struct{}

func (Router) Tag() (string, string) {
	return tagName, "Portfolio/showcase items a business posts, one per completed (or ongoing) piece of work."
}

func (Router) Routes(r *chi.Mux) {
	registerCoreRoutes(r)
	registerReviewRoutes(r)
}
