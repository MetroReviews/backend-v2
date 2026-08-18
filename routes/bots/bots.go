// Package bots exposes the /bots endpoints: Metro's own Discord bot list,
// its submission form, and the internal claim/approve/deny review pipeline
// bots go through before they're publicly listed. Handlers are split one
// concern per file; route registration mirrors that split — get/post
// registration here, review-queue action registration in
// routes_review.go, matching get.go/post.go vs action.go — this file just
// wires the two together.
package bots

import (
	"github.com/go-chi/chi/v5"
)

const tagName = "Bots"

const botColumns = `bot_id, username, banner, description, long_description, website, invite, owner, extra_owners, support, donate, library, nsfw, prefix, tags, review_note, state, avg_rating, review_count, added_at, reviewer, invite_link`

type Router struct{}

func (Router) Tag() (string, string) {
	return tagName, "Metro's own Discord bot list."
}

func (Router) Routes(r *chi.Mux) {
	registerCoreRoutes(r)
	registerReviewRoutes(r)
}
