// Package roles implements Metro's permissions system: named roles (see
// the perms package for the permissions a role can grant), assigned to
// users, that gate panel access and panel actions.
//
// A role can optionally link to a Discord server role (Role.DiscordRoleID).
// SyncMember and SyncGuild keep such a role's assignments matched to that
// Discord role's membership — the bot calls SyncMember whenever a member's
// roles change and SyncGuild for a full reconcile (on ready and via
// /syncroles), and routes/panel/callback.go calls SyncMember on login. A
// role with no linked Discord role is assigned by hand via this package's
// AssignUser/UnassignUser instead.
//
// There's no config-driven bootstrap role anymore — config.yaml's owners
// list (state.Config.IsOwner) bypasses this whole package instead (see
// SyncMember's isStaff computation and api.AuthStaff/AuthPermission), so a
// fresh deployment with an empty roles table still has someone who can log
// into the panel or use /roles in Discord to create real roles.
//
// Split by concern across files: this file holds the shared row-scanning
// plumbing, crud.go is role CRUD, assignment.go is user<->role membership
// and permission resolution, discord_link.go is queries keyed off a role's
// linked Discord role, and sync.go is the Discord membership reconciler.
package roles

import (
	"github.com/MetroReviews/backend-v2/types"
	"github.com/jackc/pgx/v5"
)

const roleColumns = `id, name, discord_role_id, permissions, created_at, updated_at`

func scanRole(row pgx.Row) (*types.Role, error) {
	var role types.Role
	if err := row.Scan(&role.ID, &role.Name, &role.DiscordRoleID, &role.Permissions, &role.CreatedAt, &role.UpdatedAt); err != nil {
		return nil, err
	}
	return &role, nil
}
