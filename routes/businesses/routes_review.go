// Review-queue and ownership-claim route registration for /businesses,
// matching review.go/claim.go. Namespaced under /review/ so it can't
// collide with the ownership-claim routes below, which use the same
// "claim" word for a completely different thing (a business owner
// claiming their business) — see review.go's package comment.
package businesses

import (
	"net/http"

	"github.com/MetroReviews/backend-v2/types"
	"github.com/go-chi/chi/v5"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
)

func registerReviewRoutes(r *chi.Mux) {
	// Review queue: the same claim -> approve/deny pipeline bots go through
	// (see the review package), surfaced together with bots in the Discord
	// /queue command.
	uapi.Route{
		Method:  uapi.POST,
		Pattern: "/businesses/{id}/review/claim",
		OpId:    "review_claim_business",
		Auth:    []uapi.AuthType{{Type: "Staff"}},
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Claim Business For Review",
				Description: "Claims a pending business for review. Requires a staff session.",
				Req:         types.Reason{},
				Resp:        types.ApiError{},
				Params: []docs.Parameter{
					{Name: "id", In: "path", Description: "The business's ID", Required: true, Schema: docs.IdSchema},
				},
			}
		},
		Handler: func(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
			return reviewAction(d, r, types.ActionClaim)
		},
	}.Route(r)

	uapi.Route{
		Method:  uapi.POST,
		Pattern: "/businesses/{id}/review/unclaim",
		OpId:    "review_unclaim_business",
		Auth:    []uapi.AuthType{{Type: "Staff"}},
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Unclaim Business",
				Description: "Releases a business back to pending. Requires a staff session.",
				Req:         types.Reason{},
				Resp:        types.ApiError{},
				Params: []docs.Parameter{
					{Name: "id", In: "path", Description: "The business's ID", Required: true, Schema: docs.IdSchema},
				},
			}
		},
		Handler: func(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
			return reviewAction(d, r, types.ActionUnclaim)
		},
	}.Route(r)

	uapi.Route{
		Method:  uapi.POST,
		Pattern: "/businesses/{id}/review/approve",
		OpId:    "review_approve_business",
		Auth:    []uapi.AuthType{{Type: "Staff"}},
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Approve Business",
				Description: "Approves a business, publishing it. Requires a staff session.",
				Req:         types.Reason{},
				Resp:        types.ApiError{},
				Params: []docs.Parameter{
					{Name: "id", In: "path", Description: "The business's ID", Required: true, Schema: docs.IdSchema},
				},
			}
		},
		Handler: func(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
			return reviewAction(d, r, types.ActionApprove)
		},
	}.Route(r)

	uapi.Route{
		Method:  uapi.POST,
		Pattern: "/businesses/{id}/review/deny",
		OpId:    "review_deny_business",
		Auth:    []uapi.AuthType{{Type: "Staff"}},
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Deny Business",
				Description: "Denies a business under review. Requires a staff session.",
				Req:         types.Reason{},
				Resp:        types.ApiError{},
				Params: []docs.Parameter{
					{Name: "id", In: "path", Description: "The business's ID", Required: true, Schema: docs.IdSchema},
				},
			}
		},
		Handler: func(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
			return reviewAction(d, r, types.ActionDeny)
		},
	}.Route(r)

	uapi.Route{
		Method:  uapi.POST,
		Pattern: "/businesses/{id}/claim",
		OpId:    "claim_business",
		Auth:    []uapi.AuthType{{Type: "User"}},
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Claim Business",
				Description: "Files an ownership claim on a business, for staff to review. Requires a logged-in user session.",
				Req:         types.ClaimCreate{},
				Resp:        types.Claim{},
				Params: []docs.Parameter{
					{Name: "id", In: "path", Description: "The business's ID", Required: true, Schema: docs.IdSchema},
				},
			}
		},
		Handler: postClaim,
	}.Route(r)

	uapi.Route{
		Method:  uapi.POST,
		Pattern: "/businesses/{id}/claims/{claim_id}/approve",
		OpId:    "approve_claim",
		Auth:    []uapi.AuthType{{Type: "Staff"}},
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Approve Claim",
				Description: "Approves an ownership claim, setting the business's verified owner. Requires a staff session.",
				Resp:        types.ApiError{},
				Params: []docs.Parameter{
					{Name: "id", In: "path", Description: "The business's ID", Required: true, Schema: docs.IdSchema},
					{Name: "claim_id", In: "path", Description: "The claim's ID", Required: true, Schema: docs.IdSchema},
				},
			}
		},
		Handler: func(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
			return resolveClaim(d, r, types.ClaimStatusApproved)
		},
	}.Route(r)

	uapi.Route{
		Method:  uapi.POST,
		Pattern: "/businesses/{id}/claims/{claim_id}/deny",
		OpId:    "deny_claim",
		Auth:    []uapi.AuthType{{Type: "Staff"}},
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Deny Claim",
				Description: "Denies an ownership claim. Requires a staff session.",
				Resp:        types.ApiError{},
				Params: []docs.Parameter{
					{Name: "id", In: "path", Description: "The business's ID", Required: true, Schema: docs.IdSchema},
					{Name: "claim_id", In: "path", Description: "The claim's ID", Required: true, Schema: docs.IdSchema},
				},
			}
		},
		Handler: func(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
			return resolveClaim(d, r, types.ClaimStatusDenied)
		},
	}.Route(r)
}
