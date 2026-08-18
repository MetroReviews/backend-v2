package businesses

import (
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
)

func updateBusiness(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	businessID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return uapi.DefaultResponse(http.StatusBadRequest)
	}

	user, resp := api.AuthUser(d.Context, r)
	if resp != nil {
		return *resp
	}

	if !user.IsStaff {
		var ownerID *uuid.UUID
		if err := state.Pool.QueryRow(d.Context,
			"SELECT owner_id FROM businesses WHERE id = $1", businessID,
		).Scan(&ownerID); err != nil {
			return helpers.InternalError(err)
		}
		if ownerID == nil || *ownerID != user.ID {
			return helpers.ErrorResponse(http.StatusForbidden, "You do not own this business")
		}
	}

	var update types.BusinessUpdate
	if hresp, ok := uapi.MarshalReq(r, &update); !ok {
		return hresp
	}

	// Column names below are all literal strings chosen by this code, never
	// derived from request input, so building the SET clause with fmt is safe.
	var setClauses []string
	var args []any
	hasUpdated := []string{}

	set := func(column string, value any, label string) {
		args = append(args, value)
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", column, len(args)))
		hasUpdated = append(hasUpdated, label)
	}

	if update.Name != nil && *update.Name != "" {
		set("name", *update.Name, "name")
	}
	if update.Description != nil {
		set("description", *update.Description, "description")
	}
	if update.Website != nil {
		set("website", *update.Website, "website")
	}
	if update.Logo != nil {
		set("logo", *update.Logo, "logo")
	}
	if update.Banner != nil {
		set("banner", *update.Banner, "banner")
	}
	if update.Address != nil {
		set("address", *update.Address, "address")
	}
	if update.City != nil {
		set("city", *update.City, "city")
	}
	if update.Country != nil {
		set("country", *update.Country, "country")
	}
	if update.Metadata != nil {
		set("metadata", update.Metadata, "metadata")
	}

	if len(setClauses) == 0 {
		return uapi.HttpResponse{Json: types.UpdatedResponse{HasUpdated: hasUpdated}}
	}

	setClauses = append(setClauses, "updated_at = NOW()")

	args = append(args, businessID)
	query := fmt.Sprintf("UPDATE businesses SET %s WHERE id = $%d", strings.Join(setClauses, ", "), len(args))
	if _, err := state.Pool.Exec(d.Context, query, args...); err != nil {
		return helpers.InternalError(err)
	}

	return uapi.HttpResponse{Json: types.UpdatedResponse{HasUpdated: hasUpdated}}
}
