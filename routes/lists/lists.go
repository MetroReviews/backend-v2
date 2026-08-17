// Package lists exposes the /list and /lists endpoints for reading and
// updating bot lists enrolled in Metro Reviews. Handlers are split one
// concern per file: this file only wires up routing.
package lists

import (
	"github.com/MetroReviews/backend-v2/types"
	"github.com/go-chi/chi/v5"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
)

const tagName = "Lists"

type Router struct{}

func (Router) Tag() (string, string) {
	return tagName, "Bot lists enrolled in Metro Reviews."
}

func (Router) Routes(r *chi.Mux) {
	uapi.Route{
		Method:  uapi.GET,
		Pattern: "/list/{id}",
		OpId:    "get_list",
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Get List",
				Description: "Gets a single bot list by its ID.",
				Resp:        types.List{},
				Params: []docs.Parameter{
					{Name: "id", In: "path", Description: "The list's ID", Required: true, Schema: docs.IdSchema},
				},
			}
		},
		Handler: getList,
	}.Route(r)

	uapi.Route{
		Method:  uapi.GET,
		Pattern: "/lists",
		OpId:    "get_all_lists",
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Get All Lists",
				Description: "Gets every bot list enrolled in Metro Reviews.",
				Resp:        []types.List{},
				RespName:    "ListArray",
			}
		},
		Handler: getAllLists,
	}.Route(r)

	uapi.Route{
		Method:  uapi.PATCH,
		Pattern: "/lists/{id}",
		OpId:    "update_list",
		Auth:    []uapi.AuthType{{Type: "List"}},
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Update List",
				Description: "Updates a list's fields. Requires the list's secret key in the Authorization header. If `reset_secret_key` is true it must be the only change requested.",
				Req:         types.ListUpdate{},
				Resp:        types.UpdatedResponse{},
				Params: []docs.Parameter{
					{Name: "id", In: "path", Description: "The list's ID", Required: true, Schema: docs.IdSchema},
				},
			}
		},
		Handler: updateList,
	}.Route(r)
}
