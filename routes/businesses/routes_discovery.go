package businesses

import (
	"github.com/MetroReviews/backend-v2/types"
	"github.com/go-chi/chi/v5"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
)

func registerDiscoveryRoutes(r *chi.Mux) {
	uapi.Route{
		Method:  uapi.GET,
		Pattern: "/businesses/{id}/similar",
		OpId:    "get_similar_businesses",
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Get Similar Businesses",
				Description: "Gets up to 6 other approved businesses in the same category, best-rated first.",
				Resp:        []types.Business{},
				RespName:    "BusinessArray",
				Params: []docs.Parameter{
					{Name: "id", In: "path", Description: "The business's ID", Required: true, Schema: docs.IdSchema},
				},
			}
		},
		Handler: getSimilarBusinesses,
	}.Route(r)

	uapi.Route{
		Method:  uapi.GET,
		Pattern: "/me/recommended",
		OpId:    "get_recommended_businesses",
		Auth:    []uapi.AuthType{{Type: "User"}},
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Get Recommended Businesses",
				Description: "Gets businesses recommended for the logged-in user, based on categories they've already reviewed (falls back to overall top-rated for a user with no review history).",
				Resp:        []types.Business{},
				RespName:    "BusinessArray",
			}
		},
		Handler: getRecommendedBusinesses,
	}.Route(r)

	uapi.Route{
		Method:  uapi.GET,
		Pattern: "/me/businesses",
		OpId:    "get_my_businesses",
		Auth:    []uapi.AuthType{{Type: "User"}},
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Get My Businesses",
				Description: "Every business the caller owns (`owner_id`) or originally submitted (`submitted_by`), in any moderation status — the listing endpoint behind an owner dashboard. Paginated with `limit`/`offset` (max limit 100).",
				Resp:        []types.Business{},
				RespName:    "BusinessArray",
				Params: []docs.Parameter{
					{Name: "offset", In: "query", Description: "Rows to skip (default 0)", Required: false, Schema: docs.IdSchema},
					{Name: "limit", In: "query", Description: "Max rows to return (default 20, max 100)", Required: false, Schema: docs.IdSchema},
				},
			}
		},
		Handler: getMyBusinesses,
	}.Route(r)

	uapi.Route{
		Method:  uapi.GET,
		Pattern: "/me/claims",
		OpId:    "get_my_claims",
		Auth:    []uapi.AuthType{{Type: "User"}},
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Get My Claims",
				Description: "Every ownership claim the caller has filed (`POST /businesses/{id}/claim`), any status, newest first — lets a dashboard show pending/approved/denied claims without polling individual businesses. Paginated with `limit`/`offset` (max limit 100).",
				Resp:        []types.Claim{},
				RespName:    "ClaimArray",
				Params: []docs.Parameter{
					{Name: "offset", In: "query", Description: "Rows to skip (default 0)", Required: false, Schema: docs.IdSchema},
					{Name: "limit", In: "query", Description: "Max rows to return (default 20, max 100)", Required: false, Schema: docs.IdSchema},
				},
			}
		},
		Handler: getMyClaims,
	}.Route(r)

	uapi.Route{
		Method:  uapi.GET,
		Pattern: "/businesses/{id}/analytics",
		OpId:    "get_business_analytics",
		Auth:    []uapi.AuthType{{Type: "User"}},
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Get Business Analytics",
				Description: "Profile view count, weekly review trend, and rating distribution. Requires the business's verified owner or a staff session.",
				Resp:        businessAnalyticsResponse{},
				Params: []docs.Parameter{
					{Name: "id", In: "path", Description: "The business's ID", Required: true, Schema: docs.IdSchema},
				},
			}
		},
		Handler: getBusinessAnalytics,
	}.Route(r)

	uapi.Route{
		Method:  uapi.GET,
		Pattern: "/businesses/{id}/widget",
		OpId:    "get_business_widget",
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Get Business Widget",
				Description: "A compact, public, cacheable payload (name, rating, up to 3 recent reviews) meant for embedding on a business's own site.",
				Resp:        widgetResponse{},
				Params: []docs.Parameter{
					{Name: "id", In: "path", Description: "The business's ID", Required: true, Schema: docs.IdSchema},
				},
			}
		},
		Handler: getBusinessWidget,
	}.Route(r)

	uapi.Route{
		Method:  uapi.PATCH,
		Pattern: "/businesses/{id}/feature",
		OpId:    "feature_business",
		Auth:    []uapi.AuthType{{Type: "Staff"}},
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Feature Business",
				Description: "Toggles a business's sponsored/featured placement. Data-model only — no payment processing. Requires the businesses.feature permission.",
				Req:         types.BusinessFeature{},
				Resp:        types.ApiError{},
				Params: []docs.Parameter{
					{Name: "id", In: "path", Description: "The business's ID", Required: true, Schema: docs.IdSchema},
				},
			}
		},
		Handler: featureBusiness,
	}.Route(r)

	uapi.Route{
		Method:  uapi.POST,
		Pattern: "/businesses/{id}/invites",
		OpId:    "post_business_invite",
		Auth:    []uapi.AuthType{{Type: "User"}},
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Invite Reviewer",
				Description: "Creates a review-invitation link for an email address, and best-effort emails it (see the mail package — no-op without SMTP configured). Requires the business's verified owner or a staff session.",
				Req:         types.ReviewInviteCreate{},
				Resp:        types.ReviewInvite{},
				Params: []docs.Parameter{
					{Name: "id", In: "path", Description: "The business's ID", Required: true, Schema: docs.IdSchema},
				},
			}
		},
		Handler: postBusinessInvite,
	}.Route(r)

	uapi.Route{
		Method:  uapi.GET,
		Pattern: "/invites/{token}",
		OpId:    "get_invite",
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Get Invite",
				Description: "Resolves a review-invitation token — public, since the token itself is the credential.",
				Resp:        types.ReviewInvite{},
				Params: []docs.Parameter{
					{Name: "token", In: "path", Description: "The invite's token", Required: true, Schema: docs.IdSchema},
				},
			}
		},
		Handler: getInvite,
	}.Route(r)
}
