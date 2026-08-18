package roles

import (
	"net/http"

	"github.com/MetroReviews/backend-v2/api"
	"github.com/MetroReviews/backend-v2/helpers"
	"github.com/MetroReviews/backend-v2/perms"
	rolesvc "github.com/MetroReviews/backend-v2/roles"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/infinitybotlist/eureka/uapi"
)

func getPermissions(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	if _, resp := api.AuthStaff(d.Context, r); resp != nil {
		return *resp
	}

	list := make([]types.Permission, len(perms.Catalog))
	for i, p := range perms.Catalog {
		list[i] = types.Permission{Slug: p.Slug, Description: p.Description}
	}
	return uapi.HttpResponse{Json: list}
}

func getMyPermissions(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	user, resp := api.AuthUser(d.Context, r)
	if resp != nil {
		return *resp
	}

	roles, err := rolesvc.UserRoles(d.Context, user.ID)
	if err != nil {
		return helpers.InternalError(err)
	}
	permissions, err := rolesvc.UserPermissions(d.Context, user.ID)
	if err != nil {
		return helpers.InternalError(err)
	}

	return uapi.HttpResponse{Json: myPermissionsResponse{Roles: roles, Permissions: permissions}}
}

func getRoles(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	if _, resp := api.AuthStaff(d.Context, r); resp != nil {
		return *resp
	}

	list, err := rolesvc.List(d.Context)
	if err != nil {
		return helpers.InternalError(err)
	}
	return uapi.HttpResponse{Json: list}
}
