package commands

import (
	"strconv"
	"strings"

	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

func HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.MessageComponentData()
	parts := strings.Split(data.CustomID, ":")
	if len(parts) < 2 {
		return
	}

	switch parts[0] {
	case "queue":
		handleQueueComponent(s, i, data, parts)
	case "role":
		handleRoleComponent(s, i, data, parts)
	}
}

func handleQueueComponent(s *discordgo.Session, i *discordgo.InteractionCreate, data discordgo.MessageComponentInteractionData, parts []string) {
	switch parts[1] {
	case "filter":
		handleQueueFilter(s, i, data, parts)
	case "page":
		handleQueuePage(s, i, parts)
	case "details":
		handleQueueDetails(s, i, data)
	case "act":
		handleQueueAction(s, i, parts)
	}
}

func handleQueueFilter(s *discordgo.Session, i *discordgo.InteractionCreate, data discordgo.MessageComponentInteractionData, parts []string) {
	if len(parts) != 3 || len(data.Values) == 0 {
		return
	}
	showAll := parts[2] == "1"
	filter := data.Values[0]
	if !validQueueFilter(filter) {
		return
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	}); err != nil {
		state.Logger.Error("[bot] failed to defer queue filter update", zap.Error(err))
		return
	}

	embed, components, err := buildQueueView(showAll, filter, 0)
	if err != nil {
		state.Logger.Error("[bot] failed to rebuild queue for filter", zap.Error(err))
		return
	}

	if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
	}); err != nil {
		state.Logger.Error("[bot] failed to edit queue filter", zap.Error(err))
	}
}

func handleQueuePage(s *discordgo.Session, i *discordgo.InteractionCreate, parts []string) {
	if len(parts) != 5 {
		return
	}
	filter := parts[2]
	if !validQueueFilter(filter) {
		return
	}
	showAll := parts[3] == "1"
	page, err := strconv.Atoi(parts[4])
	if err != nil || page < 0 {
		page = 0
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	}); err != nil {
		state.Logger.Error("[bot] failed to defer queue page update", zap.Error(err))
		return
	}

	embed, components, err := buildQueueView(showAll, filter, page)
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
	subjectType, id, ok := strings.Cut(data.Values[0], ":")
	if !ok {
		return
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Flags: discordgo.MessageFlagsEphemeral},
	}); err != nil {
		state.Logger.Error("[bot] failed to defer detail response", zap.Error(err))
		return
	}

	embed, st, err := buildDetailEmbed(subjectType, id)
	if err != nil {
		content := "Could not load that item — it may have left the queue."
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content})
		return
	}

	edit := &discordgo.WebhookEdit{Embeds: &[]*discordgo.MessageEmbed{embed}}
	if buttons := reviewButtonsFor(subjectType, id, st); len(buttons) > 0 {
		edit.Components = &buttons
	}

	if _, err := s.InteractionResponseEdit(i.Interaction, edit); err != nil {
		state.Logger.Error("[bot] failed to edit detail response", zap.Error(err))
	}
}

func handleQueueAction(s *discordgo.Session, i *discordgo.InteractionCreate, parts []string) {
	if len(parts) != 5 {
		return
	}
	subjectType := parts[2]
	actionInt, err := strconv.Atoi(parts[3])
	if err != nil {
		return
	}
	id := parts[4]
	if id == "" {
		return
	}

	openReviewModal(s, i, subjectType, types.Action(actionInt), id)
}
