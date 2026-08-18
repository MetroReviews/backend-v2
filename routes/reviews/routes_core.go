// Core review CRUD + fetching route registration, matching
// post.go/get.go/update.go. Voting/owner-response/reporting registration
// lives in routes_moderation.go, matching vote.go/response.go/report.go.
package reviews

import (
	"github.com/MetroReviews/backend-v2/types"
	"github.com/go-chi/chi/v5"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
)

func registerCoreRoutes(r *chi.Mux) {
	uapi.Route{
		Method:  uapi.POST,
		Pattern: "/reviews",
		OpId:    "post_review",
		Auth:    []uapi.AuthType{{Type: "User"}},
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Add Review",
				Description: "Posts a review against a business, a bot or a project (exactly one of `business_id`/`bot_id`/`project_id`). Requires a logged-in user session; one review per user per subject.",
				Req:         types.ReviewCreate{},
				Resp:        types.Review{},
			}
		},
		Handler: postReview,
	}.Route(r)

	uapi.Route{
		Method:  uapi.GET,
		Pattern: "/businesses/{id}/reviews",
		OpId:    "get_business_reviews",
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Get Business Reviews",
				Description: "Gets a business's published reviews, newest first. Paginated with `limit`/`offset` (max limit 100).",
				Resp:        []types.Review{},
				RespName:    "ReviewArray",
				Params: []docs.Parameter{
					{Name: "id", In: "path", Description: "The business's ID", Required: true, Schema: docs.IdSchema},
					{Name: "offset", In: "query", Description: "Rows to skip (default 0)", Required: false, Schema: docs.IdSchema},
					{Name: "limit", In: "query", Description: "Max rows to return (default 20, max 100)", Required: false, Schema: docs.IdSchema},
				},
			}
		},
		Handler: getBusinessReviews,
	}.Route(r)

	uapi.Route{
		Method:  uapi.GET,
		Pattern: "/bots/{id}/reviews",
		OpId:    "get_bot_reviews",
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Get Bot Reviews",
				Description: "Gets a bot's published reviews, newest first. Paginated with `limit`/`offset` (max limit 100).",
				Resp:        []types.Review{},
				RespName:    "ReviewArray",
				Params: []docs.Parameter{
					{Name: "id", In: "path", Description: "The bot's Discord ID", Required: true, Schema: docs.IdSchema},
					{Name: "offset", In: "query", Description: "Rows to skip (default 0)", Required: false, Schema: docs.IdSchema},
					{Name: "limit", In: "query", Description: "Max rows to return (default 20, max 100)", Required: false, Schema: docs.IdSchema},
				},
			}
		},
		Handler: getBotReviews,
	}.Route(r)

	uapi.Route{
		Method:  uapi.GET,
		Pattern: "/projects/{id}/reviews",
		OpId:    "get_project_reviews",
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Get Project Reviews",
				Description: "Gets a project's published reviews, newest first. Paginated with `limit`/`offset` (max limit 100).",
				Resp:        []types.Review{},
				RespName:    "ReviewArray",
				Params: []docs.Parameter{
					{Name: "id", In: "path", Description: "The project's ID", Required: true, Schema: docs.IdSchema},
					{Name: "offset", In: "query", Description: "Rows to skip (default 0)", Required: false, Schema: docs.IdSchema},
					{Name: "limit", In: "query", Description: "Max rows to return (default 20, max 100)", Required: false, Schema: docs.IdSchema},
				},
			}
		},
		Handler: getProjectReviews,
	}.Route(r)

	uapi.Route{
		Method:  uapi.PATCH,
		Pattern: "/reviews/{id}",
		OpId:    "update_review",
		Auth:    []uapi.AuthType{{Type: "User"}},
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Update Review",
				Description: "Edits a review. Requires the review's author.",
				Req:         types.ReviewUpdate{},
				Resp:        types.Review{},
				Params: []docs.Parameter{
					{Name: "id", In: "path", Description: "The review's ID", Required: true, Schema: docs.IdSchema},
				},
			}
		},
		Handler: updateReview,
	}.Route(r)

	uapi.Route{
		Method:  uapi.DELETE,
		Pattern: "/reviews/{id}",
		OpId:    "delete_review",
		Auth:    []uapi.AuthType{{Type: "User"}},
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Delete Review",
				Description: "Deletes a review. Requires the review's author or a staff session.",
				Resp:        types.ApiError{},
				Params: []docs.Parameter{
					{Name: "id", In: "path", Description: "The review's ID", Required: true, Schema: docs.IdSchema},
				},
			}
		},
		Handler: deleteReview,
	}.Route(r)
}
