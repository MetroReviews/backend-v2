// Package bots exposes the /bots endpoints: reading the review queue and
// adding/approving/denying/reapproving bots in it. Handlers are split one
// concern per file; this file only wires up routing.
package bots

import (
	"net/http"

	"github.com/MetroReviews/backend-v2/types"
	"github.com/go-chi/chi/v5"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
)

const tagName = "Bots"

const botColumns = `bot_id, username, banner, description, long_description, website, invite, owner, extra_owners, support, donate, library, nsfw, prefix, tags, review_note, cross_add, state, list_source, added_at, reviewer, invite_link`

type Router struct{}

func (Router) Tag() (string, string) {
	return tagName, "Bots in the review queue."
}

func (Router) Routes(r *chi.Mux) {
	uapi.Route{
		Method:  uapi.GET,
		Pattern: "/bots/{id}",
		OpId:    "get_bot",
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Get Bot",
				Description: "Gets a single queued bot by its Discord ID.",
				Resp:        types.Bot{},
				Params: []docs.Parameter{
					{Name: "id", In: "path", Description: "The bot's Discord ID", Required: true, Schema: docs.IdSchema},
				},
			}
		},
		Handler: getBot,
	}.Route(r)

	uapi.Route{
		Method:  uapi.GET,
		Pattern: "/bots",
		OpId:    "get_all_bots",
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Get All Bots",
				Description: "Gets every bot in the review queue.",
				Resp:        []types.Bot{},
				RespName:    "BotArray",
			}
		},
		Handler: getAllBots,
	}.Route(r)

	uapi.Route{
		Method:  uapi.POST,
		Pattern: "/bots",
		OpId:    "post_bots",
		Auth:    []uapi.AuthType{{Type: "List"}},
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Add Bot",
				Description: "Adds a bot to the review queue. Requires the list secret key in the Authorization header and the `list_id` query parameter. All optional fields are genuinely optional.",
				Req:         types.BotPost{},
				Resp:        types.PostBotResponse{},
				Params: []docs.Parameter{
					{Name: "list_id", In: "query", Description: "The submitting list's ID", Required: true, Schema: docs.IdSchema},
				},
			}
		},
		Handler: postBot,
	}.Route(r)

	uapi.Route{
		Method:  uapi.POST,
		Pattern: "/bots/{id}/approve",
		OpId:    "approve_bot",
		Auth:    []uapi.AuthType{{Type: "List"}},
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Approve Bot",
				Description: "Approves a bot and propagates the action to all enrolled lists. Requires the list secret key and `list_id`/`reviewer` query parameters.",
				Req:         types.Reason{},
				Resp:        types.ApiError{},
				Params: []docs.Parameter{
					{Name: "id", In: "path", Description: "The bot's Discord ID", Required: true, Schema: docs.IdSchema},
					{Name: "reviewer", In: "query", Description: "The reviewer's Discord ID", Required: true, Schema: docs.IdSchema},
					{Name: "list_id", In: "query", Description: "The list's ID", Required: true, Schema: docs.IdSchema},
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
		Auth:    []uapi.AuthType{{Type: "List"}},
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Deny Bot",
				Description: "Denies a bot and propagates the action to all enrolled lists. Requires the list secret key and `list_id`/`reviewer` query parameters.",
				Req:         types.Reason{},
				Resp:        types.ApiError{},
				Params: []docs.Parameter{
					{Name: "id", In: "path", Description: "The bot's Discord ID", Required: true, Schema: docs.IdSchema},
					{Name: "reviewer", In: "query", Description: "The reviewer's Discord ID", Required: true, Schema: docs.IdSchema},
					{Name: "list_id", In: "query", Description: "The list's ID", Required: true, Schema: docs.IdSchema},
				},
			}
		},
		Handler: func(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
			return actionBot(d, r, types.ActionDeny)
		},
	}.Route(r)

	uapi.Route{
		Method:  uapi.GET,
		Pattern: "/littlecloud/{id}",
		OpId:    "reapprove_bot",
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Reapprove Bot",
				Description: "Re-propagates an already-approved bot to all lists (used to recover from list-side errors).",
				Resp:        types.ApiError{},
				Params: []docs.Parameter{
					{Name: "id", In: "path", Description: "The bot's Discord ID", Required: true, Schema: docs.IdSchema},
				},
			}
		},
		Handler: reapproveBot,
	}.Route(r)
}
