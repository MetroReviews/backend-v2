package projects

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/MetroReviews/backend-v2/api"
	"github.com/MetroReviews/backend-v2/helpers"
	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
)

func updateProject(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	projectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return uapi.DefaultResponse(http.StatusBadRequest)
	}

	user, resp := api.AuthUser(d.Context, r)
	if resp != nil {
		return *resp
	}

	if !user.IsStaff {
		var ownerID *uuid.UUID
		err := state.Pool.QueryRow(d.Context, `
			SELECT b.owner_id FROM projects p JOIN businesses b ON b.id = p.business_id WHERE p.id = $1`, projectID,
		).Scan(&ownerID)
		if errors.Is(err, pgx.ErrNoRows) {
			return uapi.DefaultResponse(http.StatusNotFound)
		}
		if err != nil {
			return helpers.InternalError(err)
		}
		if ownerID == nil || *ownerID != user.ID {
			return helpers.ErrorResponse(http.StatusForbidden, "You do not own this project's business")
		}
	}

	var update types.ProjectUpdate
	if hresp, ok := uapi.MarshalReq(r, &update); !ok {
		return hresp
	}

	var setClauses []string
	var args []any
	hasUpdated := []string{}

	set := func(column string, value any, label string) {
		args = append(args, value)
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", column, len(args)))
		hasUpdated = append(hasUpdated, label)
	}

	if update.Title != nil && *update.Title != "" {
		set("title", *update.Title, "title")
	}
	if update.Description != nil {
		set("description", *update.Description, "description")
	}
	if update.Image != nil {
		set("image", *update.Image, "image")
	}
	if update.URL != nil {
		set("url", *update.URL, "url")
	}
	if update.CompletedAt != nil {
		set("completed_at", *update.CompletedAt, "completed_at")
	}

	if len(setClauses) == 0 {
		return uapi.HttpResponse{Json: types.UpdatedResponse{HasUpdated: hasUpdated}}
	}

	setClauses = append(setClauses, "updated_at = NOW()")

	args = append(args, projectID)
	query := fmt.Sprintf("UPDATE projects SET %s WHERE id = $%d", strings.Join(setClauses, ", "), len(args))
	if _, err := state.Pool.Exec(d.Context, query, args...); err != nil {
		return helpers.InternalError(err)
	}

	return uapi.HttpResponse{Json: types.UpdatedResponse{HasUpdated: hasUpdated}}
}
