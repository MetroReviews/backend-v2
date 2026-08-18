package projects

import (
	"errors"
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

func getProject(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	projectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return uapi.DefaultResponse(http.StatusBadRequest)
	}

	rows, err := state.Pool.Query(d.Context, "SELECT "+projectColumns+" FROM projects WHERE id = $1", projectID)
	if err != nil {
		return helpers.InternalError(err)
	}

	project, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[types.Project])
	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.DefaultResponse(http.StatusNotFound)
	}
	if err != nil {
		return helpers.InternalError(err)
	}

	if project.Status != types.StateApproved {
		staff, resp := api.AuthStaff(d.Context, r)
		if resp != nil || staff == nil {
			return uapi.DefaultResponse(http.StatusNotFound)
		}
	}

	return uapi.HttpResponse{Json: project}
}

func getBusinessProjects(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	businessID, err := uuid.Parse(chi.URLParam(r, "business_id"))
	if err != nil {
		return uapi.DefaultResponse(http.StatusBadRequest)
	}

	all := r.URL.Query().Get("all") == "true"
	if all {
		if staff, resp := api.AuthStaff(d.Context, r); resp != nil || staff == nil {
			all = false
		}
	}

	query := "SELECT " + projectColumns + " FROM projects WHERE business_id = $1"
	args := []any{businessID}
	if !all {
		query += " AND status = $2"
		args = append(args, types.StateApproved)
	}

	switch r.URL.Query().Get("sort") {
	case "rating":
		query += " ORDER BY avg_rating DESC, review_count DESC"
	case "reviews":
		query += " ORDER BY review_count DESC"
	default:
		query += " ORDER BY created_at DESC"
	}

	rows, err := state.Pool.Query(d.Context, query, args...)
	if err != nil {
		return helpers.InternalError(err)
	}

	list, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.Project])
	if err != nil {
		return helpers.InternalError(err)
	}

	return uapi.HttpResponse{Json: list}
}
