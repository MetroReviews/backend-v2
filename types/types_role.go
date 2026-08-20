package types

import (
	"time"

	"github.com/google/uuid"
)

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

type RoleUpdate struct {
	Name          *string  `json:"name" description:"New display name"`
	DiscordRoleID *string  `json:"discord_role_id" description:"New linked Discord role ID; an empty string unlinks it from Discord"`
	Permissions   []string `json:"permissions" description:"New permission slugs this role grants, replacing the existing set"`
}

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
