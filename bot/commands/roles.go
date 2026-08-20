package commands

import (
	"github.com/MetroReviews/backend-v2/state"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

const roleListLimit = 25

func cmdRoles(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !isRoleManager(i) {
		respondText(s, i, "You don't have permission to manage roles.", true)
		return
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Flags: discordgo.MessageFlagsEphemeral},
	}); err != nil {
		state.Logger.Error("[bot] failed to defer roles response", zap.Error(err))
		return
	}

	embed, components, err := buildRolesListView()
	if err != nil {
		state.Logger.Error("[bot] cmdRoles failed to load roles", zap.Error(err))
		content := "Failed to load roles."
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content})
		return
	}

	if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
	}); err != nil {
		state.Logger.Error("[bot] failed to edit deferred roles response", zap.Error(err))
	}
}
