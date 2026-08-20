package commands

import (
	"github.com/MetroReviews/backend-v2/roles"
	"github.com/MetroReviews/backend-v2/state"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

func cmdSyncRoles(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !isRoleManager(i) {
		respondText(s, i, "You don't have permission to do that.", true)
		return
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Flags: discordgo.MessageFlagsEphemeral},
	}); err != nil {
		state.Logger.Error("[bot] failed to defer syncroles response", zap.Error(err))
		return
	}

	content := "Roles synced from Discord."
	if err := roles.SyncGuild(state.Context, s, state.Config.GuildID()); err != nil {
		state.Logger.Error("[bot] cmdSyncRoles failed", zap.Error(err))
		content = "Failed to sync roles."
	}

	if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content}); err != nil {
		state.Logger.Error("[bot] failed to edit deferred syncroles response", zap.Error(err))
	}
}
