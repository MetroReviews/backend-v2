package commands

import (
	"strconv"

	"github.com/MetroReviews/backend-v2/state"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func parseDiscordID(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

func handleRoleComponent(s *discordgo.Session, i *discordgo.InteractionCreate, data discordgo.MessageComponentInteractionData, parts []string) {
	if !isRoleManager(i) {
		respondText(s, i, "You don't have permission to manage roles.", true)
		return
	}
	if len(parts) < 2 {
		return
	}

	switch parts[1] {
	case "select":
		if len(data.Values) == 0 {
			return
		}
		roleID, err := uuid.Parse(data.Values[0])
		if err != nil {
			return
		}
		showRoleDetail(s, i, roleID)
	case "backlist":
		updateWith(s, i, buildRolesListView)
	case "create":
		openRoleCreateModal(s, i)
	case "syncguild":
		handleRoleSyncGuild(s, i)
	case "rename":
		withRoleID(s, i, parts, openRoleRenameModal)
	case "permsedit":
		withRoleID(s, i, parts, showPermissionEdit)
	case "permset":
		withRoleID(s, i, parts, func(s *discordgo.Session, i *discordgo.InteractionCreate, roleID uuid.UUID) {
			applyPermissionSelection(s, i, roleID, data.Values)
		})
	case "linkstart":
		withRoleID(s, i, parts, showLinkRolePicker)
	case "linkset":
		withRoleID(s, i, parts, func(s *discordgo.Session, i *discordgo.InteractionCreate, roleID uuid.UUID) {
			applyDiscordRoleLink(s, i, roleID, data.Values)
		})
	case "unlink":
		withRoleID(s, i, parts, unlinkDiscordRole)
	case "delete":
		withRoleID(s, i, parts, showDeleteConfirm)
	case "deleteconfirm":
		withRoleID(s, i, parts, performDelete)
	case "back":
		withRoleID(s, i, parts, showRoleDetail)
	}
}

func withRoleID(s *discordgo.Session, i *discordgo.InteractionCreate, parts []string, fn func(*discordgo.Session, *discordgo.InteractionCreate, uuid.UUID)) {
	if len(parts) < 3 {
		return
	}
	roleID, err := uuid.Parse(parts[2])
	if err != nil {
		return
	}
	fn(s, i, roleID)
}

func deferUpdate(s *discordgo.Session, i *discordgo.InteractionCreate) bool {
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	}); err != nil {
		state.Logger.Error("[bot] failed to defer role component update", zap.Error(err))
		return false
	}
	return true
}

func updateWith(s *discordgo.Session, i *discordgo.InteractionCreate, build func() (*discordgo.MessageEmbed, []discordgo.MessageComponent, error)) {
	if !deferUpdate(s, i) {
		return
	}
	embed, components, err := build()
	if err != nil {
		state.Logger.Error("[bot] failed to build role view", zap.Error(err))
		content := "That role no longer exists."
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content, Embeds: &[]*discordgo.MessageEmbed{}, Components: &[]discordgo.MessageComponent{}})
		return
	}
	if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
	}); err != nil {
		state.Logger.Error("[bot] failed to edit role view", zap.Error(err))
	}
}
