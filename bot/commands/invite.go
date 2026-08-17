package commands

import (
	"strconv"

	"github.com/MetroReviews/backend-v2/helpers"
	"github.com/bwmarrin/discordgo"
)

// cmdInvite handles /invite.
func cmdInvite(s *discordgo.Session, i *discordgo.InteractionCreate, data discordgo.ApplicationCommandInteractionData) {
	botID := data.Options[0].StringValue()
	if _, err := strconv.ParseInt(botID, 10, 64); err != nil {
		respondText(s, i, "Invalid bot id", false)
		return
	}
	respondText(s, i, helpers.InviteURL(botID), false)
}
