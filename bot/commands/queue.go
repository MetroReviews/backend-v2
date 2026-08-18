// This file (plus queue_data.go and queue_view.go) implements /queue: the
// review queue's list view. Bots, businesses and projects each get their
// own queue — a select menu switches between them — rather than one
// merged list, since they went through the same claim/approve/deny
// pipeline but aren't otherwise related. queue_data.go loads and paginates
// one subject type's entries, queue_view.go renders them; this file is the
// command handler plus the shared types/constants the other two use.
package commands

import (
	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

// queuePageSize entries per /queue page. The list itself has no per-entry
// buttons (just a filter/details select plus pagination, 3 rows total at
// most), so this is only bounded by Discord's 25 select-menu options /
// embed fields.
const queuePageSize = 10

// defaultQueueFilter is which queue /queue opens to before anyone touches
// the filter select menu.
const defaultQueueFilter = "bot"

// queueFilterOrder is the fixed order the filter select menu offers the
// three queues in.
var queueFilterOrder = []string{"bot", "business", "project"}

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

var subjectEmoji = map[string]string{"bot": "🤖", "business": "🏷️", "project": "📁"}
var subjectLabel = map[string]string{"bot": "Bot", "business": "Business", "project": "Project"}
var subjectPlural = map[string]string{"bot": "Bots", "business": "Businesses", "project": "Projects"}

// queueEntry is one row of a review queue — a bot, a business or a
// project, trimmed to what the list view and its buttons need. Every entry
// in a given queueEntry slice shares the same subjectType: /queue shows
// one subject type's queue at a time (see buildFilterSelect).
type queueEntry struct {
	subjectType string // "bot", "business" or "project"
	id          string // bot_id, or a business's/project's UUID, all as strings
	name        string
	submittedBy string // pre-formatted "<@discord_id>" or a username fallback
	extra       string // bots: tags joined; businesses: category slug; projects: business name
	state       types.State
}

// cmdQueue handles /queue.
func cmdQueue(s *discordgo.Session, i *discordgo.InteractionCreate, data discordgo.ApplicationCommandInteractionData) {
	// Defer immediately: the queries below can take long enough (cold pool
	// connections, multiple round trips) to blow Discord's 3s ack window,
	// which would otherwise surface as "Unknown interaction" (10062).
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
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content}) //nolint:errcheck
		return
	}

	if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
	}); err != nil {
		state.Logger.Error("[bot] failed to edit deferred queue response", zap.Error(err))
	}
}

// buildQueueView loads filter's current queue state and renders one page
// of it. Used by /queue itself, the filter select menu, and the Prev/Next
// buttons.
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
