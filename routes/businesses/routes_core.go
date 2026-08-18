// Core CRUD route registration for /businesses — GET/POST/PATCH, matching
// get.go/post.go/update.go. Review-queue and ownership-claim route
// registration lives in businesses_review_routes.go, matching review.go/claim.go.
package businesses

import (
	"github.com/MetroReviews/backend-v2/types"
	"github.com/go-chi/chi/v5"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
)

func registerCoreRoutes(r *chi.Mux) {
	uapi.Route{
		Method:  uapi.GET,
		Pattern: "/businesses",
		OpId:    "get_all_businesses",
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Get All Businesses",
				Description: "Browses businesses. Supports `category` (category slug), `q` (name search) and `sort` (`new`, `rating`, `reviews`) query params. Anonymous callers only see approved businesses; staff also see pending/under-review/denied/suspended ones by passing `all=true`.",
				Resp:        []types.Business{},
				RespName:    "BusinessArray",
				Params: []docs.Parameter{
					{Name: "category", In: "query", Description: "Filter by category slug", Required: false, Schema: docs.IdSchema},
					{Name: "q", In: "query", Description: "Search by name", Required: false, Schema: docs.IdSchema},
					{Name: "sort", In: "query", Description: "new (default), rating, or reviews", Required: false, Schema: docs.IdSchema},
					{Name: "all", In: "query", Description: "Staff only: include non-active businesses", Required: false, Schema: docs.IdSchema},
				},
			}
		},
		Handler: getAllBusinesses,
	}.Route(r)

	uapi.Route{
		Method:  uapi.GET,
		Pattern: "/businesses/{slug}",
		OpId:    "get_business",
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Get Business",
				Description: "Gets a single business by its URL slug.",
				Resp:        types.Business{},
				Params: []docs.Parameter{
					{Name: "slug", In: "path", Description: "The business's URL slug", Required: true, Schema: docs.IdSchema},
				},
			}
		},
		Handler: getBusiness,
	}.Route(r)

	uapi.Route{
		Method:  uapi.POST,
		Pattern: "/businesses",
		OpId:    "post_business",
		Auth:    []uapi.AuthType{{Type: "User"}},
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Add Business",
				Description: "Submits a new business for review. Requires a logged-in user session. Goes through the same staff claim/approve/deny queue as bots (see /businesses/{id}/review/*) before it's publicly visible.",
				Req:         types.BusinessCreate{},
				Resp:        types.Business{},
			}
		},
		Handler: postBusiness,
	}.Route(r)

	uapi.Route{
		Method:  uapi.PATCH,
		Pattern: "/businesses/{id}",
		OpId:    "update_business",
		Auth:    []uapi.AuthType{{Type: "User"}},
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Update Business",
				Description: "Updates a business's fields. Requires the business's verified owner or a staff session.",
				Req:         types.BusinessUpdate{},
				Resp:        types.UpdatedResponse{},
				Params: []docs.Parameter{
					{Name: "id", In: "path", Description: "The business's ID", Required: true, Schema: docs.IdSchema},
				},
			}
		},
		Handler: updateBusiness,
	}.Route(r)
}
