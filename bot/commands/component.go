package commands

import (
	"strconv"
	"strings"

	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

// HandleComponent routes button/select-menu interactions produced by /queue.
func HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.MessageComponentData()
	parts := strings.Split(data.CustomID, ":")
	if len(parts) < 2 || parts[0] != "queue" {
		return
	}

	switch parts[1] {
	case "page":
		handleQueuePage(s, i, parts)
	case "details":
		handleQueueDetails(s, i, data)
	case "act":
		handleQueueAction(s, i, parts)
	}
}

func handleQueuePage(s *discordgo.Session, i *discordgo.InteractionCreate, parts []string) {
	if len(parts) != 4 {
		return
	}
	showAll := parts[2] == "1"
	page, err := strconv.Atoi(parts[3])
	if err != nil || page < 0 {
		page = 0
	}

	// Deferred *message update* edits the existing queue message in place
	// (no new "thinking" bubble), while still respecting the 3s ack window.
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	}); err != nil {
		state.Logger.Error("[bot] failed to defer queue page update", zap.Error(err))
		return
	}

	embed, components, err := buildQueueView(showAll, page)
	if err != nil {
		state.Logger.Error("[bot] failed to rebuild queue page", zap.Error(err))
		return
	}

	if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
	}); err != nil {
		state.Logger.Error("[bot] failed to edit queue page", zap.Error(err))
	}
}

func handleQueueDetails(s *discordgo.Session, i *discordgo.InteractionCreate, data discordgo.MessageComponentInteractionData) {
	if len(data.Values) == 0 {
		return
	}
	botID, err := strconv.ParseInt(data.Values[0], 10, 64)
	if err != nil {
		return
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Flags: discordgo.MessageFlagsEphemeral},
	}); err != nil {
		state.Logger.Error("[bot] failed to defer bot details response", zap.Error(err))
		return
	}

	embed, st, err := buildBotDetailEmbed(botID)
	if err != nil {
		content := "Could not load that bot — it may have left the queue."
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content}) //nolint:errcheck
		return
	}

	edit := &discordgo.WebhookEdit{Embeds: &[]*discordgo.MessageEmbed{embed}}
	if buttons := reviewButtonsFor(botID, st); len(buttons) > 0 {
		edit.Components = &buttons
	}

	if _, err := s.InteractionResponseEdit(i.Interaction, edit); err != nil {
		state.Logger.Error("[bot] failed to edit bot details response", zap.Error(err))
	}
}

func handleQueueAction(s *discordgo.Session, i *discordgo.InteractionCreate, parts []string) {
	if len(parts) != 4 {
		return
	}
	actionInt, err := strconv.Atoi(parts[2])
	if err != nil {
		return
	}
	if _, err := strconv.ParseInt(parts[3], 10, 64); err != nil {
		return
	}

	// Permission is enforced on submit (see HandleModal), matching the
	// existing /claim /approve /deny slash commands.
	openReviewModal(s, i, types.Action(actionInt), parts[3])
}
