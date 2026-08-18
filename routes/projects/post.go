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

func postProject(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	businessID, err := uuid.Parse(chi.URLParam(r, "business_id"))
	if err != nil {
		return uapi.DefaultResponse(http.StatusBadRequest)
	}

	user, resp := api.AuthUser(d.Context, r)
	if resp != nil {
		return *resp
	}

	var ownerID *uuid.UUID
	err = state.Pool.QueryRow(d.Context,
		"SELECT owner_id FROM businesses WHERE id = $1", businessID,
	).Scan(&ownerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.DefaultResponse(http.StatusNotFound)
	}
	if err != nil {
		return helpers.InternalError(err)
	}
	if !user.IsStaff && (ownerID == nil || *ownerID != user.ID) {
		return helpers.ErrorResponse(http.StatusForbidden, "You do not own this business")
	}

	var payload types.ProjectCreate
	if hresp, ok := uapi.MarshalReq(r, &payload); !ok {
		return hresp
	}

	rows, err := state.Pool.Query(d.Context, `
		INSERT INTO projects (business_id, title, description, image, url, completed_at, submitted_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING `+projectColumns,
		businessID, payload.Title, payload.Description, payload.Image, payload.URL, payload.CompletedAt, user.ID,
	)
	if err != nil {
		return helpers.InternalError(err)
	}

	project, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[types.Project])
	if err != nil {
		return helpers.InternalError(err)
	}

	return uapi.HttpResponse{Json: project}
}
