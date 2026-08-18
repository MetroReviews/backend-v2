package types

import (
	"time"

	"github.com/google/uuid"
)

// Role is a named bundle of permissions (see the perms package for the
// catalog) that can be assigned to users, granting them panel access and
// whatever else the permissions cover. A role with a non-nil
// DiscordRoleID has its membership kept in sync with that Discord server
// role by the bot (see the roles package); one with a nil DiscordRoleID is
// panel-only and assigned by hand.
type Role struct {
	ID            uuid.UUID `db:"id" json:"id" description:"The role's ID"`
	Name          string    `db:"name" json:"name" description:"The role's display name"`
	DiscordRoleID *int64    `db:"discord_role_id" json:"discord_role_id" description:"The linked Discord role ID, if this role's membership is synced from the guild"`
	Permissions   []string  `db:"permissions" json:"permissions" description:"Permission slugs this role grants; \"*\" grants everything"`
	CreatedAt     time.Time `db:"created_at" json:"created_at" description:"When the role was created"`
	UpdatedAt     time.Time `db:"updated_at" json:"updated_at" description:"When the role was last updated"`
}

type RoleCreate struct {
	Name          string   `json:"name" validate:"required" msg:"A name is required" description:"The role's display name"`
	DiscordRoleID *string  `json:"discord_role_id" description:"A Discord role ID to sync this role's membership from, if any"`
	Permissions   []string `json:"permissions" description:"Permission slugs this role grants"`
}

// RoleUpdate patches a role. DiscordRoleID follows the usual "nil means
// leave unchanged" rule, plus one wrinkle: an empty string unlinks the
// role from Discord (there's no other way to express "remove" with a
// pointer-to-string field).
type RoleUpdate struct {
	Name          *string  `json:"name" description:"New display name"`
	DiscordRoleID *string  `json:"discord_role_id" description:"New linked Discord role ID; an empty string unlinks it from Discord"`
	Permissions   []string `json:"permissions" description:"New permission slugs this role grants, replacing the existing set"`
}

// Permission is one entry in Metro's fixed permission catalog (see the
// perms package), for populating a role editor.
type Permission struct {
	Slug        string `json:"slug" description:"The permission's slug, as stored on a role"`
	Description string `json:"description" description:"What the permission grants"`
}

type TeamMember struct {
	Username    string   `json:"username" description:"The member's username"`
	ID          string   `json:"id" description:"The member's Discord ID"`
	Avatar      string   `json:"avatar" description:"The member's avatar URL"`
	IsListOwner bool     `json:"is_list_owner" description:"Whether the member holds the List Owner role"`
	Sudo        bool     `json:"sudo" description:"Whether the member holds the Sudo role"`
	Roles       []string `json:"roles" description:"List-related role names the member holds"`
}
