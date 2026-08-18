package roles

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/MetroReviews/backend-v2/api"
	"github.com/MetroReviews/backend-v2/helpers"
	"github.com/MetroReviews/backend-v2/perms"
	"github.com/MetroReviews/backend-v2/rpc"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
)

// invalidPermissions returns every slug in requested that isn't in perms'
// catalog (or the wildcard), for rejecting typos up front rather than
// silently storing a permission nothing ever checks for.
func invalidPermissions(requested []string) []string {
	var bad []string
	for _, p := range requested {
		if !perms.Valid(p) {
			bad = append(bad, p)
		}
	}
	return bad
}

// forbiddenResponse maps an *rpc.ForbiddenError to the same 403 shape
// api.AuthPermission used to produce directly; every other error is the
// caller's to handle.
func forbiddenResponse(err error) (uapi.HttpResponse, bool) {
	var forbidden *rpc.ForbiddenError
	if errors.As(err, &forbidden) {
		return helpers.ErrorResponse(http.StatusForbidden, "Missing required permission: "+forbidden.Permission), true
	}
	return uapi.HttpResponse{}, false
}

func createRole(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	user, resp := api.AuthUser(d.Context, r)
	if resp != nil {
		return *resp
	}

	var payload types.RoleCreate
	if hresp, ok := uapi.MarshalReq(r, &payload); !ok {
		return hresp
	}

	if bad := invalidPermissions(payload.Permissions); len(bad) > 0 {
		return helpers.ErrorResponse(http.StatusBadRequest, "Unknown permission(s): "+strings.Join(bad, ", "))
	}

	var discordRoleID *int64
	if payload.DiscordRoleID != nil && *payload.DiscordRoleID != "" {
		id, err := strconv.ParseInt(*payload.DiscordRoleID, 10, 64)
		if err != nil {
			return helpers.ErrorResponse(http.StatusBadRequest, "discord_role_id must be a valid Discord ID")
		}
		discordRoleID = &id
	}

	actor := rpc.Actor{UserID: user.ID, DiscordID: user.DiscordID, Source: "api"}
	role, err := rpc.CreateRole(d.Context, actor, payload.Name, discordRoleID, payload.Permissions)
	if hresp, ok := forbiddenResponse(err); ok {
		return hresp
	}
	if err != nil {
		return helpers.InternalError(err)
	}
	return uapi.HttpResponse{Json: role}
}

func updateRole(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	user, resp := api.AuthUser(d.Context, r)
	if resp != nil {
		return *resp
	}

	roleID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return uapi.DefaultResponse(http.StatusBadRequest)
	}

	var payload types.RoleUpdate
	if hresp, ok := uapi.MarshalReq(r, &payload); !ok {
		return hresp
	}

	if bad := invalidPermissions(payload.Permissions); len(bad) > 0 {
		return helpers.ErrorResponse(http.StatusBadRequest, "Unknown permission(s): "+strings.Join(bad, ", "))
	}

	var discordRoleID *int64
	unlink := false
	if payload.DiscordRoleID != nil {
		if *payload.DiscordRoleID == "" {
			unlink = true
		} else {
			id, err := strconv.ParseInt(*payload.DiscordRoleID, 10, 64)
			if err != nil {
				return helpers.ErrorResponse(http.StatusBadRequest, "discord_role_id must be a valid Discord ID")
			}
			discordRoleID = &id
		}
	}

	actor := rpc.Actor{UserID: user.ID, DiscordID: user.DiscordID, Source: "api"}
	role, err := rpc.UpdateRole(d.Context, actor, roleID, payload.Name, discordRoleID, unlink, payload.Permissions)
	if hresp, ok := forbiddenResponse(err); ok {
		return hresp
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.DefaultResponse(http.StatusNotFound)
	}
	if err != nil {
		return helpers.InternalError(err)
	}
	return uapi.HttpResponse{Json: role}
}

func deleteRole(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	user, resp := api.AuthUser(d.Context, r)
	if resp != nil {
		return *resp
	}

	roleID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return uapi.DefaultResponse(http.StatusBadRequest)
	}

	actor := rpc.Actor{UserID: user.ID, DiscordID: user.DiscordID, Source: "api"}
	if err := rpc.DeleteRole(d.Context, actor, roleID); err != nil {
		if hresp, ok := forbiddenResponse(err); ok {
			return hresp
		}
		return helpers.InternalError(err)
	}
	return uapi.HttpResponse{Json: types.ApiError{Message: "Role deleted", Error: false}}
}
