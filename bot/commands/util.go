package commands

import (
	"strconv"

	"github.com/MetroReviews/backend-v2/helpers"
	"github.com/MetroReviews/backend-v2/perms"
	"github.com/MetroReviews/backend-v2/roles"
	"github.com/MetroReviews/backend-v2/state"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

func row(c discordgo.MessageComponent) discordgo.MessageComponent {
	return discordgo.ActionsRow{Components: []discordgo.MessageComponent{c}}
}

func respondText(s *discordgo.Session, i *discordgo.InteractionCreate, content string, ephemeral bool) {
	data := &discordgo.InteractionResponseData{Content: content}
	if ephemeral {
		data.Flags = discordgo.MessageFlagsEphemeral
	}
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: data,
	}); err != nil {
		state.Logger.Error("[bot] failed to respond to interaction", zap.Error(err))
	}
}

func modalValues(data discordgo.ModalSubmitInteractionData) map[string]string {
	out := map[string]string{}
	for _, comp := range data.Components {
		ar, ok := comp.(*discordgo.ActionsRow)
		if !ok {
			continue
		}
		for _, inner := range ar.Components {
			if ti, ok := inner.(*discordgo.TextInput); ok {
				out[ti.CustomID] = ti.Value
			}
		}
	}
	return out
}

func interactionUserID(i *discordgo.InteractionCreate) int64 {
	var idStr string
	if i.Member != nil && i.Member.User != nil {
		idStr = i.Member.User.ID
	} else if i.User != nil {
		idStr = i.User.ID
	}
	id, _ := strconv.ParseInt(idStr, 10, 64)
	return id
}

func interactionUsername(i *discordgo.InteractionCreate) string {
	if i.Member != nil && i.Member.User != nil {
		return i.Member.User.Username
	}
	if i.User != nil {
		return i.User.Username
	}
	return ""
}

func isReviewer(i *discordgo.InteractionCreate) bool {
	return hasDiscordPermission(i, perms.QueueReview)
}

func isRoleManager(i *discordgo.InteractionCreate) bool {
	return hasDiscordPermission(i, perms.RolesManage)
}

func hasDiscordPermission(i *discordgo.InteractionCreate, want string) bool {
	if state.Config.IsOwner(interactionUserID(i)) {
		return true
	}
	if i.Member == nil {
		return false
	}

	roleIDs, err := roles.DiscordRoleIDsWithPermission(state.Context, want)
	if err != nil {
		state.Logger.Error("[bot] failed to load role IDs for permission check", zap.Error(err), zap.String("permission", want))
		return false
	}

	for _, roleID := range i.Member.Roles {
		rid, err := strconv.ParseInt(roleID, 10, 64)
		if err != nil {
			continue
		}
		if helpers.Contains(roleIDs, rid) {
			return true
		}
	}
	return false
}
