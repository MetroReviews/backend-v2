package reviews

import (
	"github.com/go-chi/chi/v5"
)

const tagName = "Reviews"

const reviewColumns = `id, business_id, project_id, author_id, rating, title, body, owner_response, owner_response_at, helpful_count, status, created_at, updated_at, photos, flag_reason, verified`

type Router struct{}

func (Router) Tag() (string, string) {
	return tagName, "Reviews and ratings, shared by businesses and projects."
}

func (Router) Routes(r *chi.Mux) {
	registerCoreRoutes(r)
	registerModerationRoutes(r)
}
