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

func DeleteRole(ctx context.Context, actor Actor, id uuid.UUID) error {
	if err := authorize(ctx, actor, perms.RolesManage); err != nil {
		return err
	}

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

func SyncGuildRoles(ctx context.Context, actor Actor) error {
	if err := authorize(ctx, actor, perms.RolesManage); err != nil {
		return err
	}
	if state.Discord == nil {
		return ErrDiscordUnavailable
	}
	return roles.SyncGuild(ctx, state.Discord, state.Config.GuildID())
}

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
