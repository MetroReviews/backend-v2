package commands

import (
	"github.com/MetroReviews/backend-v2/roles"
	"github.com/MetroReviews/backend-v2/state"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func showRoleDetail(s *discordgo.Session, i *discordgo.InteractionCreate, roleID uuid.UUID) {
	updateWith(s, i, func() (*discordgo.MessageEmbed, []discordgo.MessageComponent, error) {
		return buildRoleDetailView(roleID)
	})
}

func showPermissionEdit(s *discordgo.Session, i *discordgo.InteractionCreate, roleID uuid.UUID) {
	updateWith(s, i, func() (*discordgo.MessageEmbed, []discordgo.MessageComponent, error) {
		role, err := roles.Get(state.Context, roleID)
		if err != nil {
			return nil, nil, err
		}
		embed, components := buildPermissionEditView(role)
		return embed, components, nil
	})
}

func showLinkRolePicker(s *discordgo.Session, i *discordgo.InteractionCreate, roleID uuid.UUID) {
	updateWith(s, i, func() (*discordgo.MessageEmbed, []discordgo.MessageComponent, error) {
		role, err := roles.Get(state.Context, roleID)
		if err != nil {
			return nil, nil, err
		}
		embed, components := buildLinkRolePickerView(role)
		return embed, components, nil
	})
}

func showDeleteConfirm(s *discordgo.Session, i *discordgo.InteractionCreate, roleID uuid.UUID) {
	updateWith(s, i, func() (*discordgo.MessageEmbed, []discordgo.MessageComponent, error) {
		role, err := roles.Get(state.Context, roleID)
		if err != nil {
			return nil, nil, err
		}
		count, err := roles.MemberCount(state.Context, roleID)
		if err != nil {
			return nil, nil, err
		}
		embed, components := buildDeleteConfirmView(role, count)
		return embed, components, nil
	})
}

func applyPermissionSelection(s *discordgo.Session, i *discordgo.InteractionCreate, roleID uuid.UUID, selected []string) {

	permissions := selected
	for _, p := range selected {
		if p == "*" {
			permissions = []string{"*"}
			break
		}
	}
	if permissions == nil {
		permissions = []string{}
	}

	if _, err := roles.Update(state.Context, roleID, nil, nil, false, permissions); err != nil {
		state.Logger.Error("[bot] failed to update role permissions", zap.Error(err))
	}
	showRoleDetail(s, i, roleID)
}

func applyDiscordRoleLink(s *discordgo.Session, i *discordgo.InteractionCreate, roleID uuid.UUID, values []string) {
	if !deferUpdate(s, i) {
		return
	}

	if len(values) != 0 {
		if discordRoleID, err := parseDiscordID(values[0]); err == nil {
			if _, err := roles.Update(state.Context, roleID, nil, &discordRoleID, false, nil); err != nil {
				if isUniqueViolation(err) {
					followupError(s, i, "That Discord role is already linked to another role.")
				} else {
					state.Logger.Error("[bot] failed to link discord role", zap.Error(err))
				}
			}
		}
	}

	redrawRoleDetail(s, i, roleID)
}

func redrawRoleDetail(s *discordgo.Session, i *discordgo.InteractionCreate, roleID uuid.UUID) {
	embed, components, err := buildRoleDetailView(roleID)
	if err != nil {
		state.Logger.Error("[bot] failed to build role detail view", zap.Error(err))
		return
	}
	if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
	}); err != nil {
		state.Logger.Error("[bot] failed to edit role view", zap.Error(err))
	}
}

func unlinkDiscordRole(s *discordgo.Session, i *discordgo.InteractionCreate, roleID uuid.UUID) {
	if _, err := roles.Update(state.Context, roleID, nil, nil, true, nil); err != nil {
		state.Logger.Error("[bot] failed to unlink discord role", zap.Error(err))
	}
	showRoleDetail(s, i, roleID)
}

func performDelete(s *discordgo.Session, i *discordgo.InteractionCreate, roleID uuid.UUID) {
	if !deferUpdate(s, i) {
		return
	}
	if err := roles.Delete(state.Context, roleID); err != nil {
		state.Logger.Error("[bot] failed to delete role", zap.Error(err))
	}
	embed, components, err := buildRolesListView()
	if err != nil {
		state.Logger.Error("[bot] failed to rebuild roles list after delete", zap.Error(err))
		return
	}
	if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
	}); err != nil {
		state.Logger.Error("[bot] failed to edit role view after delete", zap.Error(err))
	}
}

func handleRoleSyncGuild(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !deferUpdate(s, i) {
		return
	}
	content := "Synced roles from Discord."
	if state.Discord == nil {
		content = "The Discord bot isn't connected."
	} else if err := roles.SyncGuild(state.Context, state.Discord, state.Config.GuildID()); err != nil {
		state.Logger.Error("[bot] role sync from /roles failed", zap.Error(err))
		content = "Failed to sync roles."
	}

	embed, components, err := buildRolesListView()
	if err != nil {
		state.Logger.Error("[bot] failed to rebuild roles list after sync", zap.Error(err))
		return
	}
	if embed.Description != "" {
		content += "\n" + embed.Description
	}
	embed.Description = content
	if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
	}); err != nil {
		state.Logger.Error("[bot] failed to edit role view after sync", zap.Error(err))
	}
}
