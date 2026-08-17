package lists

import (
	"errors"
	"net/http"

	"github.com/MetroReviews/backend-v2/helpers"
	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
)

func getList(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return uapi.DefaultResponse(http.StatusBadRequest)
	}

	rows, err := state.Pool.Query(d.Context,
		"SELECT id, name, description, domain, state, icon FROM bot_list WHERE id = $1", id)
	if err != nil {
		return helpers.InternalError(err)
	}

	list, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[types.List])
	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.DefaultResponse(http.StatusNotFound)
	}
	if err != nil {
		return helpers.InternalError(err)
	}

	return uapi.HttpResponse{Json: list}
}

func getAllLists(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	rows, err := state.Pool.Query(d.Context,
		"SELECT id, name, description, domain, state, icon FROM bot_list ORDER BY id ASC")
	if err != nil {
		return helpers.InternalError(err)
	}

	lists, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.List])
	if err != nil {
		return helpers.InternalError(err)
	}

	return uapi.HttpResponse{Json: lists}
}
