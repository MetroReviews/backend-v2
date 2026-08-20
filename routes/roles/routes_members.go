package roles

import (
	"github.com/MetroReviews/backend-v2/perms"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/go-chi/chi/v5"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
)

func registerMemberRoutes(r *chi.Mux) {
	uapi.Route{
		Method:  uapi.PUT,
		Pattern: "/roles/{id}/members/{user_id}",
		OpId:    "assign_role",
		Auth:    []uapi.AuthType{{Type: "Staff"}},
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Assign Role",
				Description: "Grants a role to a user. For a Discord-linked role this is overwritten by the next sync unless they also hold the Discord role — assign that instead. Requires the `" + perms.RolesManage + "` permission.",
				Resp:        types.ApiError{},
				Params: []docs.Parameter{
					{Name: "id", In: "path", Description: "The role's ID", Required: true, Schema: docs.IdSchema},
					{Name: "user_id", In: "path", Description: "The Metro user ID to grant the role to", Required: true, Schema: docs.IdSchema},
				},
			}
		},
		Handler: assignRole,
	}.Route(r)

	uapi.Route{
		Method:  uapi.DELETE,
		Pattern: "/roles/{id}/members/{user_id}",
		OpId:    "unassign_role",
		Auth:    []uapi.AuthType{{Type: "Staff"}},
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Unassign Role",
				Description: "Revokes a role from a user. Requires the `" + perms.RolesManage + "` permission.",
				Resp:        types.ApiError{},
				Params: []docs.Parameter{
					{Name: "id", In: "path", Description: "The role's ID", Required: true, Schema: docs.IdSchema},
					{Name: "user_id", In: "path", Description: "The Metro user ID to revoke the role from", Required: true, Schema: docs.IdSchema},
				},
			}
		},
		Handler: unassignRole,
	}.Route(r)

	uapi.Route{
		Method:  uapi.POST,
		Pattern: "/roles/sync",
		OpId:    "sync_roles",
		Auth:    []uapi.AuthType{{Type: "Staff"}},
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Sync Roles",
				Description: "Re-syncs every Discord-linked role's assignments from the guild's current membership. The bot already does this on ready and whenever a member's roles change; use this after linking/unlinking a role without waiting for the next such event. Requires the `" + perms.RolesManage + "` permission.",
				Resp:        types.ApiError{},
			}
		},
		Handler: syncRoles,
	}.Route(r)
}
