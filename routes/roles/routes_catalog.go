// Permission catalog + role CRUD route registration, matching get.go/crud.go.
// Membership/sync registration lives in routes_members.go, matching
// members.go/sync.go.
package roles

import (
	"github.com/MetroReviews/backend-v2/perms"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/go-chi/chi/v5"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
)

func registerRoleRoutes(r *chi.Mux) {
	uapi.Route{
		Method:  uapi.GET,
		Pattern: "/permissions",
		OpId:    "get_permissions",
		Auth:    []uapi.AuthType{{Type: "Staff"}},
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Get Permissions",
				Description: "Returns the fixed permission catalog roles can be granted from.",
				Resp:        []types.Permission{},
				RespName:    "PermissionArray",
			}
		},
		Handler: getPermissions,
	}.Route(r)

	uapi.Route{
		Method:  uapi.GET,
		Pattern: "/me/permissions",
		OpId:    "get_my_permissions",
		Auth:    []uapi.AuthType{{Type: "User"}},
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Get My Permissions",
				Description: "Returns the calling user's roles and effective permissions.",
				Resp:        myPermissionsResponse{},
			}
		},
		Handler: getMyPermissions,
	}.Route(r)

	uapi.Route{
		Method:  uapi.GET,
		Pattern: "/roles",
		OpId:    "get_roles",
		Auth:    []uapi.AuthType{{Type: "Staff"}},
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Get Roles",
				Description: "Returns every role.",
				Resp:        []types.Role{},
				RespName:    "RoleArray",
			}
		},
		Handler: getRoles,
	}.Route(r)

	uapi.Route{
		Method:  uapi.POST,
		Pattern: "/roles",
		OpId:    "create_role",
		Auth:    []uapi.AuthType{{Type: "Staff"}},
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Create Role",
				Description: "Creates a role. Requires the `" + perms.RolesManage + "` permission.",
				Req:         types.RoleCreate{},
				Resp:        types.Role{},
			}
		},
		Handler: createRole,
	}.Route(r)

	uapi.Route{
		Method:  uapi.PATCH,
		Pattern: "/roles/{id}",
		OpId:    "update_role",
		Auth:    []uapi.AuthType{{Type: "Staff"}},
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Update Role",
				Description: "Patches a role's name, linked Discord role and/or permissions. Requires the `" + perms.RolesManage + "` permission.",
				Req:         types.RoleUpdate{},
				Resp:        types.Role{},
				Params: []docs.Parameter{
					{Name: "id", In: "path", Description: "The role's ID", Required: true, Schema: docs.IdSchema},
				},
			}
		},
		Handler: updateRole,
	}.Route(r)

	uapi.Route{
		Method:  uapi.DELETE,
		Pattern: "/roles/{id}",
		OpId:    "delete_role",
		Auth:    []uapi.AuthType{{Type: "Staff"}},
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Delete Role",
				Description: "Deletes a role and every user's assignment to it. Requires the `" + perms.RolesManage + "` permission.",
				Resp:        types.ApiError{},
				Params: []docs.Parameter{
					{Name: "id", In: "path", Description: "The role's ID", Required: true, Schema: docs.IdSchema},
				},
			}
		},
		Handler: deleteRole,
	}.Route(r)
}
