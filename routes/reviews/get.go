package reviews

import (
	"net/http"

	"github.com/MetroReviews/backend-v2/api"
	"github.com/MetroReviews/backend-v2/helpers"
	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
)

func getBusinessReviews(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	businessID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return uapi.DefaultResponse(http.StatusBadRequest)
	}

	limit, offset := helpers.Pagination(r, 20, 100)

	rows, err := state.Pool.Query(d.Context,
		"SELECT "+reviewColumns+" FROM reviews WHERE business_id = $1 AND status = $2 ORDER BY created_at DESC LIMIT $3 OFFSET $4",
		businessID, types.ReviewStatusPublished, limit, offset)
	if err != nil {
		return helpers.InternalError(err)
	}

	list, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.Review])
	if err != nil {
		return helpers.InternalError(err)
	}

	return uapi.HttpResponse{Json: list}
}

func getMyReviews(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	user, resp := api.AuthUser(d.Context, r)
	if resp != nil {
		return *resp
	}

	limit, offset := helpers.Pagination(r, 20, 100)

	rows, err := state.Pool.Query(d.Context,
		"SELECT "+reviewColumns+" FROM reviews WHERE author_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3",
		user.ID, limit, offset)
	if err != nil {
		return helpers.InternalError(err)
	}

	list, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.Review])
	if err != nil {
		return helpers.InternalError(err)
	}

	return uapi.HttpResponse{Json: list}
}

func getProjectReviews(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	projectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return uapi.DefaultResponse(http.StatusBadRequest)
	}

	limit, offset := helpers.Pagination(r, 20, 100)

	rows, err := state.Pool.Query(d.Context,
		"SELECT "+reviewColumns+" FROM reviews WHERE project_id = $1 AND status = $2 ORDER BY created_at DESC LIMIT $3 OFFSET $4",
		projectID, types.ReviewStatusPublished, limit, offset)
	if err != nil {
		return helpers.InternalError(err)
	}

	list, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.Review])
	if err != nil {
		return helpers.InternalError(err)
	}

	return uapi.HttpResponse{Json: list}
}
