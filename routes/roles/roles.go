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
