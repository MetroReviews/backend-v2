package bots

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/MetroReviews/backend-v2/api"
	"github.com/MetroReviews/backend-v2/helpers"
	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/go-chi/chi/v5"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
)

func getBot(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		return uapi.DefaultResponse(http.StatusBadRequest)
	}

	rows, err := state.Pool.Query(d.Context, "SELECT "+botColumns+" FROM bots WHERE bot_id = $1", id)
	if err != nil {
		return helpers.InternalError(err)
	}

	b, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[types.Bot])
	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.DefaultResponse(http.StatusNotFound)
	}
	if err != nil {
		return helpers.InternalError(err)
	}

	// Bots still in the review queue are only visible to staff and (once
	// per-user filtering exists) their owner — for now, staff only.
	if b.State != types.StateApproved && !isStaffCaller(d.Context, r) {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	return uapi.HttpResponse{Json: b}
}

func getAllBots(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	query := "SELECT " + botColumns + " FROM bots"
	if !isStaffCaller(d.Context, r) {
		query += " WHERE state = " + strconv.Itoa(int(types.StateApproved))
	}
	query += " ORDER BY bot_id ASC"

	rows, err := state.Pool.Query(d.Context, query)
	if err != nil {
		return helpers.InternalError(err)
	}

	list, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.Bot])
	if err != nil {
		return helpers.InternalError(err)
	}

	return uapi.HttpResponse{Json: list}
}

// isStaffCaller reports whether the request carries a valid staff session,
// without failing the request when it doesn't — these are public endpoints
// that simply show more to staff than to anonymous callers.
func isStaffCaller(ctx context.Context, r *http.Request) bool {
	u, resp := api.AuthStaff(ctx, r)
	return resp == nil && u != nil
}
