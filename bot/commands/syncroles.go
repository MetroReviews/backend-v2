package commands

import (
	"github.com/MetroReviews/backend-v2/roles"
	"github.com/MetroReviews/backend-v2/state"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

// cmdSyncRoles handles /syncroles: a full reconcile of every Discord-linked
// role's assignments against the guild's current membership (see
// roles.SyncGuild). The bot already does this on ready and incrementally
// whenever a member's roles change (bot/bot.go); this is for right after
// linking/unlinking a role from the panel, without waiting for the next
// such event. Unlike /sync (Discord command registration, open to any
// reviewer), this changes who has staff/panel permissions, so it's gated
// to roles.manage instead of queue.review.
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
