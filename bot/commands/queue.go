package commands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/MetroReviews/backend-v2/helpers"
	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

// queuePageSize bots per /queue page. The list itself has no per-bot
// buttons (just a details/action select + pagination, 2 rows total), so
// this is only bounded by Discord's 25 select-menu options / embed fields.
const queuePageSize = 10

var stateNames = map[types.State]string{
	types.StatePending:     "PENDING",
	types.StateUnderReview: "UNDER_REVIEW",
	types.StateApproved:    "APPROVED",
	types.StateDenied:      "DENIED",
}

var stateEmoji = map[types.State]string{
	types.StatePending:     "🟡",
	types.StateUnderReview: "🔵",
	types.StateApproved:    "🟢",
	types.StateDenied:      "🔴",
}

// queueEntry is one row of the bot review queue, trimmed to what the list
// view and its buttons need.
type queueEntry struct {
	botID    int64
	username string
	owner    int64
	tags     []string
	state    types.State
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

	embed, components, err := buildQueueView(showAll, 0)
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

// fetchQueueEntries loads the bots to show: pending/under_review only,
// unless showAll requests every state.
func fetchQueueEntries(showAll bool) ([]queueEntry, error) {
	query := "SELECT bot_id, username, owner, tags, state FROM bot_queue"
	var args []any
	if !showAll {
		query += " WHERE state = $1 OR state = $2"
		args = append(args, types.StatePending, types.StateUnderReview)
	}
	query += " ORDER BY state ASC, bot_id ASC"

	rows, err := state.Pool.Query(state.Context, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []queueEntry
	for rows.Next() {
		var e queueEntry
		if err := rows.Scan(&e.botID, &e.username, &e.owner, &e.tags, &e.state); err != nil {
			continue
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func queuePageCount(total int) int {
	if total == 0 {
		return 1
	}
	return (total + queuePageSize - 1) / queuePageSize
}

func queuePageSlice(entries []queueEntry, page int) []queueEntry {
	start := page * queuePageSize
	if start >= len(entries) {
		return nil
	}
	end := min(start+queuePageSize, len(entries))
	return entries[start:end]
}

// buildQueueView loads the current queue state and renders one page of it.
// Used both by /queue itself and by the Prev/Next buttons.
func buildQueueView(showAll bool, page int) (*discordgo.MessageEmbed, []discordgo.MessageComponent, error) {
	entries, err := fetchQueueEntries(showAll)
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

	embed := buildQueueEmbed(entries, showAll, page, totalPages)
	components := buildQueueComponents(entries, showAll, page, totalPages)
	return embed, components, nil
}

func queueFilterLabel(showAll bool) string {
	if showAll {
		return "all states"
	}
	return "pending & under review"
}

func buildQueueEmbed(entries []queueEntry, showAll bool, page, totalPages int) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title: "🤖 Bot Review Queue",
		Color: 0x5865f2, // blurple
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Page %d/%d • %d bot(s) shown • %s", page+1, totalPages, len(entries), queueFilterLabel(showAll)),
		},
	}

	if len(entries) == 0 {
		embed.Description = "The queue is empty."
		return embed
	}

	for _, e := range entries {
		if e.state == types.StatePending {
			embed.Color = 0xf1c40f // gold: something needs claiming
			break
		}
	}

	for _, e := range queuePageSlice(entries, page) {
		name := fmt.Sprintf("%s %s — %s", stateEmoji[e.state], e.username, stateNames[e.state])
		value := fmt.Sprintf("ID `%d` • Owner <@%d>", e.botID, e.owner)
		if len(e.tags) > 0 {
			value += "\nTags: " + strings.Join(e.tags, ", ")
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: name, Value: value})
	}
	return embed
}

// reviewButtonsFor returns the action buttons for a bot's current state
// (empty for states that have nothing left to do), shown on its detail
// panel after it's picked from the "select a bot" menu.
func reviewButtonsFor(botID int64, st types.State) []discordgo.MessageComponent {
	customID := func(action types.Action) string { return fmt.Sprintf("queue:act:%d:%d", action, botID) }

	switch st {
	case types.StatePending:
		return []discordgo.MessageComponent{
			discordgo.ActionsRow{Components: []discordgo.MessageComponent{
				discordgo.Button{Label: "Claim", Style: discordgo.SuccessButton, CustomID: customID(types.ActionClaim)},
			}},
		}
	case types.StateUnderReview:
		return []discordgo.MessageComponent{
			discordgo.ActionsRow{Components: []discordgo.MessageComponent{
				discordgo.Button{Label: "Approve", Style: discordgo.SuccessButton, CustomID: customID(types.ActionApprove)},
				discordgo.Button{Label: "Deny", Style: discordgo.DangerButton, CustomID: customID(types.ActionDeny)},
				discordgo.Button{Label: "Unclaim", Style: discordgo.SecondaryButton, CustomID: customID(types.ActionUnclaim)},
			}},
		}
	default:
		return nil
	}
}

// buildDetailsSelect builds the "select a bot" menu, scoped to whatever
// page is currently on screen (always well under Discord's 25-option cap
// given queuePageSize). Picking an option opens that bot's detail panel
// with its Claim/Approve/Deny/Unclaim buttons attached.
func buildDetailsSelect(pageEntries []queueEntry) discordgo.SelectMenu {
	opts := make([]discordgo.SelectMenuOption, 0, len(pageEntries))
	for _, e := range pageEntries {
		opts = append(opts, discordgo.SelectMenuOption{
			Label:       helpers.Truncate(e.username, 100),
			Value:       strconv.FormatInt(e.botID, 10),
			Description: helpers.Truncate(fmt.Sprintf("%s • ID %d • Owner %d", stateNames[e.state], e.botID, e.owner), 100),
		})
	}

	return discordgo.SelectMenu{
		CustomID:    "queue:details",
		Placeholder: "📋 Select a bot to manage…",
		Options:     opts,
	}
}

// buildQueueComponents lays out the select-a-bot menu plus pagination —
// per-bot actions live on the detail panel opened from the select menu,
// not inline in the list.
func buildQueueComponents(entries []queueEntry, showAll bool, page, totalPages int) []discordgo.MessageComponent {
	var comps []discordgo.MessageComponent

	pageEntries := queuePageSlice(entries, page)
	if len(pageEntries) > 0 {
		comps = append(comps, discordgo.ActionsRow{Components: []discordgo.MessageComponent{buildDetailsSelect(pageEntries)}})
	}

	if totalPages > 1 {
		showAllFlag := "0"
		if showAll {
			showAllFlag = "1"
		}
		comps = append(comps, discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "◀ Prev",
				Style:    discordgo.SecondaryButton,
				CustomID: fmt.Sprintf("queue:page:%s:%d", showAllFlag, page-1),
				Disabled: page <= 0,
			},
			discordgo.Button{
				Label:    fmt.Sprintf("Page %d/%d", page+1, totalPages),
				Style:    discordgo.SecondaryButton,
				CustomID: "queue:noop",
				Disabled: true,
			},
			discordgo.Button{
				Label:    "Next ▶",
				Style:    discordgo.SecondaryButton,
				CustomID: fmt.Sprintf("queue:page:%s:%d", showAllFlag, page+1),
				Disabled: page >= totalPages-1,
			},
		}})
	}

	return comps
}
