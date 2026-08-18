// Review-queue route registration for /projects, matching review.go.
package projects

import (
	"net/http"

	"github.com/MetroReviews/backend-v2/types"
	"github.com/go-chi/chi/v5"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
)

func registerReviewRoutes(r *chi.Mux) {
	// The same claim -> approve/deny pipeline bots and businesses go
	// through (see the review package), surfaced together with them in
	// the Discord /queue command.
	uapi.Route{
		Method:  uapi.POST,
		Pattern: "/projects/{id}/review/claim",
		OpId:    "review_claim_project",
		Auth:    []uapi.AuthType{{Type: "Staff"}},
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Claim Project For Review",
				Description: "Claims a pending project for review. Requires a staff session.",
				Req:         types.Reason{},
				Resp:        types.ApiError{},
				Params: []docs.Parameter{
					{Name: "id", In: "path", Description: "The project's ID", Required: true, Schema: docs.IdSchema},
				},
			}
		},
		Handler: func(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
			return reviewAction(d, r, types.ActionClaim)
		},
	}.Route(r)

	uapi.Route{
		Method:  uapi.POST,
		Pattern: "/projects/{id}/review/unclaim",
		OpId:    "review_unclaim_project",
		Auth:    []uapi.AuthType{{Type: "Staff"}},
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Unclaim Project",
				Description: "Releases a project back to pending. Requires a staff session.",
				Req:         types.Reason{},
				Resp:        types.ApiError{},
				Params: []docs.Parameter{
					{Name: "id", In: "path", Description: "The project's ID", Required: true, Schema: docs.IdSchema},
				},
			}
		},
		Handler: func(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
			return reviewAction(d, r, types.ActionUnclaim)
		},
	}.Route(r)

	uapi.Route{
		Method:  uapi.POST,
		Pattern: "/projects/{id}/review/approve",
		OpId:    "review_approve_project",
		Auth:    []uapi.AuthType{{Type: "Staff"}},
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Approve Project",
				Description: "Approves a project, publishing it. Requires a staff session.",
				Req:         types.Reason{},
				Resp:        types.ApiError{},
				Params: []docs.Parameter{
					{Name: "id", In: "path", Description: "The project's ID", Required: true, Schema: docs.IdSchema},
				},
			}
		},
		Handler: func(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
			return reviewAction(d, r, types.ActionApprove)
		},
	}.Route(r)

	uapi.Route{
		Method:  uapi.POST,
		Pattern: "/projects/{id}/review/deny",
		OpId:    "review_deny_project",
		Auth:    []uapi.AuthType{{Type: "Staff"}},
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Deny Project",
				Description: "Denies a project under review. Requires a staff session.",
				Req:         types.Reason{},
				Resp:        types.ApiError{},
				Params: []docs.Parameter{
					{Name: "id", In: "path", Description: "The project's ID", Required: true, Schema: docs.IdSchema},
				},
			}
		},
		Handler: func(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
			return reviewAction(d, r, types.ActionDeny)
		},
	}.Route(r)
}
