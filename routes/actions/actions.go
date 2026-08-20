package actions

import (
	"net/http"
	"strconv"

	"github.com/MetroReviews/backend-v2/helpers"
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
	return tagName, "Review queue action history (claim/unclaim/approve/deny), for both businesses and projects."
}

func (Router) Routes(r *chi.Mux) {
	uapi.Route{
		Method:  uapi.GET,
		Pattern: "/actions",
		OpId:    "get_actions",
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary: "Get Actions",
				Description: `Returns a list of review queue actions (claim/unclaim/approve/deny) taken on businesses and projects. Filter by ` + "`target_type`" + ` (` + "`business`" + ` or ` + "`project`" + `).

Paginated using ` + "`limit`" + ` (max rows to return) and ` + "`offset`" + ` (rows to skip). Maximum limit is 200.`,
				Resp:     []types.ModerationAction{},
				RespName: "ModerationActionArray",
				Params: []docs.Parameter{
					{Name: "target_type", In: "query", Description: "Filter to business or project actions only", Required: false, Schema: docs.IdSchema},
					{Name: "offset", In: "query", Description: "Rows to skip (default 0)", Required: false, Schema: docs.IdSchema},
					{Name: "limit", In: "query", Description: "Max rows to return (default 50, max 200)", Required: false, Schema: docs.IdSchema},
				},
			}
		},
		Handler: getActions,
	}.Route(r)
}

func getActions(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	limit, offset := helpers.Pagination(r, 50, 200)

	targetType := r.URL.Query().Get("target_type")

	query := "SELECT id, target_type, target_id, action, reason, reviewer, action_time FROM moderation_actions"
	args := []any{}
	if targetType == "business" || targetType == "project" {
		args = append(args, targetType)
		query += " WHERE target_type = $1"
	}
	query += " ORDER BY action_time DESC LIMIT $" + strconv.Itoa(len(args)+1) + " OFFSET $" + strconv.Itoa(len(args)+2)
	args = append(args, limit, offset)

	rows, err := state.Pool.Query(d.Context, query, args...)
	if err != nil {
		return helpers.InternalError(err)
	}

	list, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.ModerationAction])
	if err != nil {
		return helpers.InternalError(err)
	}

	return uapi.HttpResponse{Json: list}
}
