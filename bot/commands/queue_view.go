package commands

import (
	"fmt"
	"strings"

	"github.com/MetroReviews/backend-v2/helpers"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/bwmarrin/discordgo"
)

func queueFilterLabel(showAll bool) string {
	if showAll {
		return "all states"
	}
	return "pending & under review"
}

func buildQueueEmbed(entries []queueEntry, filter string, showAll bool, page, totalPages int) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("%s Review Queue — %s", subjectEmoji[filter], subjectPlural[filter]),
		Color: 0x5865f2, // blurple
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Page %d/%d • %d item(s) shown • %s", page+1, totalPages, len(entries), queueFilterLabel(showAll)),
		},
	}

	if len(entries) == 0 {
		embed.Description = fmt.Sprintf("No %s in the queue.", strings.ToLower(subjectPlural[filter]))
		return embed
	}

	for _, e := range entries {
		if e.state == types.StatePending {
			embed.Color = 0xf1c40f // gold: something needs claiming
			break
		}
	}

	for _, e := range queuePageSlice(entries, page) {
		name := fmt.Sprintf("%s %s — %s", stateEmoji[e.state], e.name, stateNames[e.state])
		value := fmt.Sprintf("ID `%s` • Submitted by %s", e.id, e.submittedBy)
		if e.extra != "" {
			value += "\n" + e.extra
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: name, Value: value})
	}
	return embed
}

// reviewButtonsFor returns the action buttons for an entry's current state
// (empty for states that have nothing left to do), shown on its detail
// panel after it's picked from the "select an item" menu.
func reviewButtonsFor(subjectType, id string, st types.State) []discordgo.MessageComponent {
	customID := func(action types.Action) string { return fmt.Sprintf("queue:act:%s:%d:%s", subjectType, action, id) }

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

// buildFilterSelect builds the "which queue" menu that switches /queue
// between bots, businesses and projects — showAll rides along in its
// CustomID so switching queues doesn't reset that choice, the same way
// the pagination buttons carry the current filter.
func buildFilterSelect(filter string, showAll bool) discordgo.SelectMenu {
	showAllFlag := "0"
	if showAll {
		showAllFlag = "1"
	}

	opts := make([]discordgo.SelectMenuOption, 0, len(queueFilterOrder))
	for _, subjectType := range queueFilterOrder {
		opts = append(opts, discordgo.SelectMenuOption{
			Label:   subjectPlural[subjectType],
			Value:   subjectType,
			Emoji:   &discordgo.ComponentEmoji{Name: subjectEmoji[subjectType]},
			Default: subjectType == filter,
		})
	}

	return discordgo.SelectMenu{
		CustomID:    "queue:filter:" + showAllFlag,
		Placeholder: "📋 Choose a queue…",
		Options:     opts,
	}
}

// buildDetailsSelect builds the "select an item" menu, scoped to whatever
// page is currently on screen (always well under Discord's 25-option cap
// given queuePageSize). Picking an option opens that item's detail panel
// with its Claim/Approve/Deny/Unclaim buttons attached.
func buildDetailsSelect(pageEntries []queueEntry) discordgo.SelectMenu {
	opts := make([]discordgo.SelectMenuOption, 0, len(pageEntries))
	for _, e := range pageEntries {
		opts = append(opts, discordgo.SelectMenuOption{
			Label:       helpers.Truncate(e.name, 100),
			Value:       e.subjectType + ":" + e.id,
			Description: helpers.Truncate(fmt.Sprintf("%s • ID %s • Submitted by %s", stateNames[e.state], e.id, e.submittedBy), 100),
		})
	}

	return discordgo.SelectMenu{
		CustomID:    "queue:details",
		Placeholder: "Select an item to manage…",
		Options:     opts,
	}
}

// buildQueueComponents lays out the filter and select-an-item menus plus
// pagination — per-entry actions live on the detail panel opened from the
// select menu, not inline in the list.
func buildQueueComponents(entries []queueEntry, filter string, showAll bool, page, totalPages int) []discordgo.MessageComponent {
	comps := []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{buildFilterSelect(filter, showAll)}},
	}

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
				CustomID: fmt.Sprintf("queue:page:%s:%s:%d", filter, showAllFlag, page-1),
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
				CustomID: fmt.Sprintf("queue:page:%s:%s:%d", filter, showAllFlag, page+1),
				Disabled: page >= totalPages-1,
			},
		}})
	}

	return comps
}
