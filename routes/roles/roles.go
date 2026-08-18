// Package roles exposes the panel's permissions system over HTTP: the
// permission catalog, role CRUD, assigning/unassigning roles to users, and
// a manual trigger for the Discord role sync that the bot otherwise runs
// on its own (see the roles domain package). Handlers are split one
// concern per file; route registration mirrors that split — permission
// catalog/role CRUD here, membership/sync registration in
// routes_members.go — this file just wires the two together.
package roles

import (
	"github.com/MetroReviews/backend-v2/types"
	"github.com/go-chi/chi/v5"
)

const tagName = "Roles"

type Router struct{}

func (Router) Tag() (string, string) {
	return tagName, "Panel roles and permissions."
}

func (Router) Routes(r *chi.Mux) {
	registerRoleRoutes(r)
	registerMemberRoutes(r)
}

type myPermissionsResponse struct {
	Roles       []types.Role `json:"roles" description:"The roles the caller holds"`
	Permissions []string     `json:"permissions" description:"The caller's effective permissions (the union of every role held)"`
}
