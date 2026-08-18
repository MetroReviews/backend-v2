package rpc

import (
	"context"
	"fmt"
	"time"

	"github.com/MetroReviews/backend-v2/perms"
	"github.com/MetroReviews/backend-v2/roles"
	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// CreateRole creates a role on actor's behalf, enforcing roles.manage.
// Errors from roles.Create (e.g. a duplicate name) are returned unwrapped
// so callers can still check them with errors.As/isUniqueViolation.
func CreateRole(ctx context.Context, actor Actor, name string, discordRoleID *int64, permissions []string) (*types.Role, error) {
	if err := authorize(ctx, actor, perms.RolesManage); err != nil {
		return nil, err
	}
	role, err := roles.Create(ctx, name, discordRoleID, permissions)
	if err != nil {
		return nil, err
	}
	announceRoleChange(actor, "Created", role.Name)
	return role, nil
}

// UpdateRole patches a role on actor's behalf, enforcing roles.manage.
func UpdateRole(ctx context.Context, actor Actor, id uuid.UUID, name *string, discordRoleID *int64, unlinkDiscordRole bool, permissions []string) (*types.Role, error) {
	if err := authorize(ctx, actor, perms.RolesManage); err != nil {
		return nil, err
	}
	role, err := roles.Update(ctx, id, name, discordRoleID, unlinkDiscordRole, permissions)
	if err != nil {
		return nil, err
	}
	announceRoleChange(actor, "Updated", role.Name)
	return role, nil
}

// DeleteRole deletes a role on actor's behalf, enforcing roles.manage.
func DeleteRole(ctx context.Context, actor Actor, id uuid.UUID) error {
	if err := authorize(ctx, actor, perms.RolesManage); err != nil {
		return err
	}
	// Look the role up first purely so the audit log can name it — a
	// best-effort lookup, not required for the delete itself to succeed.
	role, _ := roles.Get(ctx, id)
	if err := roles.Delete(ctx, id); err != nil {
		return err
	}
	name := id.String()
	if role != nil {
		name = role.Name
	}
	announceRoleChange(actor, "Deleted", name)
	return nil
}

// AssignRole grants roleID to userID on actor's behalf, enforcing
// roles.manage.
func AssignRole(ctx context.Context, actor Actor, userID, roleID uuid.UUID) error {
	if err := authorize(ctx, actor, perms.RolesManage); err != nil {
		return err
	}
	if err := roles.AssignUser(ctx, userID, roleID); err != nil {
		return err
	}
	announceRoleChange(actor, "Assigned", fmt.Sprintf("role %s to user %s", roleID, userID))
	return nil
}

// UnassignRole revokes roleID from userID on actor's behalf, enforcing
// roles.manage.
func UnassignRole(ctx context.Context, actor Actor, userID, roleID uuid.UUID) error {
	if err := authorize(ctx, actor, perms.RolesManage); err != nil {
		return err
	}
	if err := roles.UnassignUser(ctx, userID, roleID); err != nil {
		return err
	}
	announceRoleChange(actor, "Unassigned", fmt.Sprintf("role %s from user %s", roleID, userID))
	return nil
}

// SyncGuildRoles re-syncs every Discord-linked role's assignments on
// actor's behalf, enforcing roles.manage. Returns ErrDiscordUnavailable
// if the bot isn't connected — the same precondition regardless of
// whether the sync was triggered over HTTP or from Discord itself.
func SyncGuildRoles(ctx context.Context, actor Actor) error {
	if err := authorize(ctx, actor, perms.RolesManage); err != nil {
		return err
	}
	if state.Discord == nil {
		return ErrDiscordUnavailable
	}
	return roles.SyncGuild(ctx, state.Discord, state.Config.GuildID())
}

// announceRoleChange posts a short record of a role CRUD/assignment
// change to the configured logs channel — a no-op if Discord isn't
// connected or no logs channel is configured, same guard
// roles.postRoleSyncLog uses for the sync-specific log it already sends.
func announceRoleChange(actor Actor, action, detail string) {
	if state.Discord == nil || state.Config.LogsChannelID() == 0 {
		return
	}

	embed := &discordgo.MessageEmbed{
		Title: "🔑 Role " + action,
		Color: 0x5865f2,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "By", Value: mentionActor(actor), Inline: true},
			{Name: "Via", Value: actor.Source, Inline: true},
			{Name: "Detail", Value: detail},
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	channelID := fmt.Sprintf("%d", state.Config.LogsChannelID())
	if _, err := state.Discord.ChannelMessageSendEmbed(channelID, embed); err != nil {
		state.Logger.Error("[rpc] failed to send role change log", zap.Error(err))
	}
}
