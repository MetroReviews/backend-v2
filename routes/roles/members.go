package roles

import (
	"net/http"

	"github.com/MetroReviews/backend-v2/api"
	"github.com/MetroReviews/backend-v2/helpers"
	"github.com/MetroReviews/backend-v2/rpc"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/infinitybotlist/eureka/uapi"
)

func roleAndUserID(r *http.Request) (roleID, userID uuid.UUID, ok bool) {
	roleID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return uuid.Nil, uuid.Nil, false
	}
	userID, err = uuid.Parse(chi.URLParam(r, "user_id"))
	if err != nil {
		return uuid.Nil, uuid.Nil, false
	}
	return roleID, userID, true
}

func assignRole(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	user, resp := api.AuthUser(d.Context, r)
	if resp != nil {
		return *resp
	}

	roleID, userID, ok := roleAndUserID(r)
	if !ok {
		return uapi.DefaultResponse(http.StatusBadRequest)
	}

	actor := rpc.Actor{UserID: user.ID, DiscordID: user.DiscordID, Source: "api"}
	if err := rpc.AssignRole(d.Context, actor, userID, roleID); err != nil {
		if hresp, ok := forbiddenResponse(err); ok {
			return hresp
		}
		return helpers.InternalError(err)
	}
	return uapi.HttpResponse{Json: types.ApiError{Message: "Role assigned", Error: false}}
}

func unassignRole(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	user, resp := api.AuthUser(d.Context, r)
	if resp != nil {
		return *resp
	}

	roleID, userID, ok := roleAndUserID(r)
	if !ok {
		return uapi.DefaultResponse(http.StatusBadRequest)
	}

	actor := rpc.Actor{UserID: user.ID, DiscordID: user.DiscordID, Source: "api"}
	if err := rpc.UnassignRole(d.Context, actor, userID, roleID); err != nil {
		if hresp, ok := forbiddenResponse(err); ok {
			return hresp
		}
		return helpers.InternalError(err)
	}
	return uapi.HttpResponse{Json: types.ApiError{Message: "Role unassigned", Error: false}}
}
