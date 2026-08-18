// GET/POST route registration for /bots, matching get.go/post.go.
// Review-queue action registration lives in routes_review.go, matching
// action.go.
package bots

import (
	"github.com/MetroReviews/backend-v2/types"
	"github.com/go-chi/chi/v5"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
)

func registerCoreRoutes(r *chi.Mux) {
	uapi.Route{
		Method:  uapi.GET,
		Pattern: "/bots/{id}",
		OpId:    "get_bot",
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Get Bot",
				Description: "Gets a single bot by its Discord ID.",
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
				Description: "Gets every bot on the list. Anonymous callers only see approved bots; staff see every state.",
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
		Auth:    []uapi.AuthType{{Type: "User"}},
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Add Bot",
				Description: "Submits a bot to the review queue. Requires a logged-in user session.",
				Req:         types.BotPost{},
				Resp:        types.PostBotResponse{},
			}
		},
		Handler: postBot,
	}.Route(r)
}
