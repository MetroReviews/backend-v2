// Review-queue action route registration for /bots, matching action.go.
// GET/POST registration lives in routes_core.go.
package bots

import (
	"net/http"

	"github.com/MetroReviews/backend-v2/types"
	"github.com/go-chi/chi/v5"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
)

func registerReviewRoutes(r *chi.Mux) {
	uapi.Route{
		Method:  uapi.POST,
		Pattern: "/bots/{id}/claim",
		OpId:    "claim_bot",
		Auth:    []uapi.AuthType{{Type: "Staff"}},
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Claim Bot",
				Description: "Claims a pending bot for review. Requires a staff session.",
				Req:         types.Reason{},
				Resp:        types.ApiError{},
				Params: []docs.Parameter{
					{Name: "id", In: "path", Description: "The bot's Discord ID", Required: true, Schema: docs.IdSchema},
				},
			}
		},
		Handler: func(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
			return actionBot(d, r, types.ActionClaim)
		},
	}.Route(r)

	uapi.Route{
		Method:  uapi.POST,
		Pattern: "/bots/{id}/unclaim",
		OpId:    "unclaim_bot",
		Auth:    []uapi.AuthType{{Type: "Staff"}},
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Unclaim Bot",
				Description: "Releases a bot back to pending. Requires a staff session.",
				Req:         types.Reason{},
				Resp:        types.ApiError{},
				Params: []docs.Parameter{
					{Name: "id", In: "path", Description: "The bot's Discord ID", Required: true, Schema: docs.IdSchema},
				},
			}
		},
		Handler: func(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
			return actionBot(d, r, types.ActionUnclaim)
		},
	}.Route(r)

	uapi.Route{
		Method:  uapi.POST,
		Pattern: "/bots/{id}/approve",
		OpId:    "approve_bot",
		Auth:    []uapi.AuthType{{Type: "Staff"}},
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Approve Bot",
				Description: "Approves a bot, publishing it to the public bot list. Requires a staff session.",
				Req:         types.Reason{},
				Resp:        types.ApiError{},
				Params: []docs.Parameter{
					{Name: "id", In: "path", Description: "The bot's Discord ID", Required: true, Schema: docs.IdSchema},
				},
			}
		},
		Handler: func(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
			return actionBot(d, r, types.ActionApprove)
		},
	}.Route(r)

	uapi.Route{
		Method:  uapi.POST,
		Pattern: "/bots/{id}/deny",
		OpId:    "deny_bot",
		Auth:    []uapi.AuthType{{Type: "Staff"}},
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Deny Bot",
				Description: "Denies a bot under review. Requires a staff session.",
				Req:         types.Reason{},
				Resp:        types.ApiError{},
				Params: []docs.Parameter{
					{Name: "id", In: "path", Description: "The bot's Discord ID", Required: true, Schema: docs.IdSchema},
				},
			}
		},
		Handler: func(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
			return actionBot(d, r, types.ActionDeny)
		},
	}.Route(r)
}
