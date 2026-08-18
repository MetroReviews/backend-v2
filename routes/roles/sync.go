package roles

import (
	"net/http"

	"github.com/MetroReviews/backend-v2/api"
	"github.com/MetroReviews/backend-v2/helpers"
	"github.com/MetroReviews/backend-v2/perms"
	rolesvc "github.com/MetroReviews/backend-v2/roles"
	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/infinitybotlist/eureka/uapi"
)

func syncRoles(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	if _, resp := api.AuthPermission(d.Context, r, perms.RolesManage); resp != nil {
		return *resp
	}

	if state.Discord == nil {
		return helpers.ErrorResponse(http.StatusServiceUnavailable, "The Discord bot isn't connected")
	}

	if err := rolesvc.SyncGuild(d.Context, state.Discord, state.Config.GuildID()); err != nil {
		return helpers.InternalError(err)
	}
	return uapi.HttpResponse{Json: types.ApiError{Message: "Roles synced", Error: false}}
}
