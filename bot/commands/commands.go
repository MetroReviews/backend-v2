// Package commands holds every Discord bot command: the slash commands,
// the /queue review UI (buttons/selects/modals it drives), and the legacy
// `%`-prefixed text commands. One file per command/feature; this file only
// declares the command list and dispatches interactions to them.
package commands

import (
	"strconv"

	"github.com/MetroReviews/backend-v2/types"
	"github.com/bwmarrin/discordgo"
)

// reviewTypeOption is the "type" choice (bot, business or project) attached
// to the standalone /claim /unclaim /approve /deny commands, since —
// unlike picking an item from /queue — there's no other way to know which
// table the given ID belongs to.
var reviewTypeOption = &discordgo.ApplicationCommandOption{
	Type:        discordgo.ApplicationCommandOptionString,
	Name:        "type",
	Description: "What you're reviewing",
	Required:    true,
	Choices: []*discordgo.ApplicationCommandOptionChoice{
		{Name: "Bot", Value: "bot"},
		{Name: "Business", Value: "business"},
		{Name: "Project", Value: "project"},
	},
}

// Definitions is the bot's guild slash command list, registered on ready
// (see Sync, called from bot.go's onReady).
var Definitions = []*discordgo.ApplicationCommand{
	{
		Name:        "queue",
		Description: "Show the review queue (bots, businesses and projects)",
		Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionBoolean, Name: "show_all", Description: "Show all states", Required: false},
		},
	},
	{Name: "syncroles", Description: "Re-syncs panel roles/permissions from this server's Discord roles"},
	{Name: "roles", Description: "Create, edit and delete panel roles"},
	{Name: "support", Description: "Show the link to support"},
	{Name: "claim", Description: "Claim a bot, business or project for review", Options: []*discordgo.ApplicationCommandOption{reviewTypeOption}},
	{Name: "unclaim", Description: "Unclaim a bot, business or project", Options: []*discordgo.ApplicationCommandOption{reviewTypeOption}},
	{Name: "approve", Description: "Approve a bot, business or project", Options: []*discordgo.ApplicationCommandOption{reviewTypeOption}},
	{Name: "deny", Description: "Deny a bot, business or project", Options: []*discordgo.ApplicationCommandOption{reviewTypeOption}},
}

// Sync bulk-overwrites the guild's slash commands with Definitions.
func Sync(s *discordgo.Session, guildID uint64) error {
	gid := strconv.FormatUint(guildID, 10)
	_, err := s.ApplicationCommandBulkOverwrite(s.State.User.ID, gid, Definitions)
	return err
}

// HandleCommand dispatches a top-level slash command to its handler.
func HandleCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ApplicationCommandData()
	switch data.Name {
	case "queue":
		cmdQueue(s, i, data)
	case "syncroles":
		cmdSyncRoles(s, i)
	case "roles":
		cmdRoles(s, i)
	case "support":
		respondText(s, i, "https://github.com/MetroReviews/support", false)
	case "claim":
		openReviewModal(s, i, reviewType(data), types.ActionClaim, "")
	case "unclaim":
		openReviewModal(s, i, reviewType(data), types.ActionUnclaim, "")
	case "approve":
		openReviewModal(s, i, reviewType(data), types.ActionApprove, "")
	case "deny":
		openReviewModal(s, i, reviewType(data), types.ActionDeny, "")
	}
}

// reviewType reads the "type" option shared by /claim /unclaim /approve
// /deny, defaulting to "bot" if somehow missing (Discord enforces Required,
// so this is just belt-and-suspenders).
func reviewType(data discordgo.ApplicationCommandInteractionData) string {
	for _, opt := range data.Options {
		if opt.Name == "type" {
			return opt.StringValue()
		}
	}
	return "bot"
}
