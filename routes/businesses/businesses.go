package businesses

import (
	"github.com/MetroReviews/backend-v2/helpers"
	"github.com/go-chi/chi/v5"
)

const maxGalleryPhotos = 12

func validateGallery(urls []string) (out []string, ok bool) {
	return helpers.ValidateImageURLs(urls, maxGalleryPhotos)
}

const tagName = "Businesses"

const businessColumns = `id, category_id, slug, name, description, website, logo, banner, address, city, country, metadata, owner_id, submitted_by, status, reviewer, avg_rating, review_count, created_at, updated_at, latitude, longitude, gallery, featured, featured_until, view_count`

type Router struct{}

func (Router) Tag() (string, string) {
	return tagName, "Reviewable businesses: any service or business on Metro."
}

func (Router) Routes(r *chi.Mux) {
	registerCoreRoutes(r)
	registerReviewRoutes(r)
	registerDiscoveryRoutes(r)
}
