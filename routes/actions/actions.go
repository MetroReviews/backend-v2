package actions

import (
	"net/http"
	"strconv"

	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/go-chi/chi/v5"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
)

const tagName = "Actions"

type Router struct{}

func (Router) Tag() (string, string) {
	return tagName, "Review action history (claim/unclaim/approve/deny)."
}

func (Router) Routes(r *chi.Mux) {
	uapi.Route{
		Method:  uapi.GET,
		Pattern: "/actions",
		OpId:    "get_actions",
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary: "Get Actions",
				Description: `Returns a list of review actions (claim/unclaim/approve/deny).

` + "`list_source`" + ` will not be present in all cases.

**This is purely to allow Metro Review lists to debug their code.**

Paginated using ` + "`limit`" + ` (max rows to return) and ` + "`offset`" + ` (rows to skip). Maximum limit is 200.`,
				Resp:     []types.ActionRow{},
				RespName: "ActionArray",
				Params: []docs.Parameter{
					{Name: "offset", In: "query", Description: "Rows to skip (default 0)", Required: false, Schema: docs.IdSchema},
					{Name: "limit", In: "query", Description: "Max rows to return (default 50, max 200)", Required: false, Schema: docs.IdSchema},
				},
			}
		},
		Handler: getActions,
	}.Route(r)
}

func getActions(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	offset := int64(0)
	limit := int64(50)

	if v := r.URL.Query().Get("offset"); v != "" {
		if p, err := strconv.ParseInt(v, 10, 64); err == nil {
			offset = p
		}
	}
	if v := r.URL.Query().Get("limit"); v != "" {
		if p, err := strconv.ParseInt(v, 10, 64); err == nil {
			limit = p
		}
	}

	if limit > 200 {
		return uapi.HttpResponse{Json: []types.ActionRow{}}
	}

	rows, err := state.Pool.Query(d.Context,
		"SELECT id, bot_id, action, reason, reviewer, action_time, list_source FROM bot_action ORDER BY action_time DESC LIMIT $1 OFFSET $2",
		limit, offset)
	if err != nil {
		return uapi.HttpResponse{Status: http.StatusInternalServerError, Json: types.ApiError{Message: err.Error(), Error: true}}
	}

	list, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.ActionRow])
	if err != nil {
		return uapi.HttpResponse{Status: http.StatusInternalServerError, Json: types.ApiError{Message: err.Error(), Error: true}}
	}

	return uapi.HttpResponse{Json: list}
}
