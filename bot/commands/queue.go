package commands

import (
	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

const queuePageSize = 10

const defaultQueueFilter = "business"

var queueFilterOrder = []string{"business", "project"}

func validQueueFilter(s string) bool {
	for _, f := range queueFilterOrder {
		if f == s {
			return true
		}
	}
	return false
}

var stateNames = map[types.State]string{
	types.StatePending:     "PENDING",
	types.StateUnderReview: "UNDER_REVIEW",
	types.StateApproved:    "APPROVED",
	types.StateDenied:      "DENIED",
	types.StateSuspended:   "SUSPENDED",
}

var stateEmoji = map[types.State]string{
	types.StatePending:     "🟡",
	types.StateUnderReview: "🔵",
	types.StateApproved:    "🟢",
	types.StateDenied:      "🔴",
	types.StateSuspended:   "🟣",
}

var subjectEmoji = map[string]string{"business": "🏷️", "project": "📁"}
var subjectLabel = map[string]string{"business": "Business", "project": "Project"}
var subjectPlural = map[string]string{"business": "Businesses", "project": "Projects"}

type queueEntry struct {
	subjectType string
	id          string
	name        string
	submittedBy string
	extra       string
	state       types.State
}

func cmdQueue(s *discordgo.Session, i *discordgo.InteractionCreate, data discordgo.ApplicationCommandInteractionData) {

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	}); err != nil {
		state.Logger.Error("[bot] failed to defer queue response", zap.Error(err))
		return
	}

	showAll := false
	for _, opt := range data.Options {
		if opt.Name == "show_all" {
			showAll = opt.BoolValue()
		}
	}

	embed, components, err := buildQueueView(showAll, defaultQueueFilter, 0)
	if err != nil {
		state.Logger.Error("[bot] cmdQueue failed to load entries", zap.Error(err))
		content := "Failed to load the queue."
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content})
		return
	}

	if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
	}); err != nil {
		state.Logger.Error("[bot] failed to edit deferred queue response", zap.Error(err))
	}
}

func buildQueueView(showAll bool, filter string, page int) (*discordgo.MessageEmbed, []discordgo.MessageComponent, error) {
	entries, err := fetchQueueEntries(showAll, filter)
	if err != nil {
		return nil, nil, err
	}

	totalPages := queuePageCount(len(entries))
	if page < 0 {
		page = 0
	}
	if page > totalPages-1 {
		page = totalPages - 1
	}

	embed := buildQueueEmbed(entries, filter, showAll, page, totalPages)
	components := buildQueueComponents(entries, filter, showAll, page, totalPages)
	return embed, components, nil
}
