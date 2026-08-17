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

// Definitions is the bot's guild slash command list, registered on ready
// and by /sync.
var Definitions = []*discordgo.ApplicationCommand{
	{
		Name:        "invite",
		Description: "Get a bot's invite link",
		Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionString, Name: "bot_id", Description: "The bot's ID", Required: true},
		},
	},
	{
		Name:        "queue",
		Description: "Show the bot queue",
		Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionBoolean, Name: "show_all", Description: "Show all states", Required: false},
		},
	},
	{Name: "sync", Description: "Syncs all commands"},
	{Name: "support", Description: "Show the link to support"},
	{Name: "claim", Description: "Claim a bot"},
	{Name: "unclaim", Description: "Unclaim a bot"},
	{Name: "approve", Description: "Approve a bot"},
	{Name: "deny", Description: "Deny a bot"},
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
	case "invite":
		cmdInvite(s, i, data)
	case "queue":
		cmdQueue(s, i, data)
	case "sync":
		cmdSync(s, i)
	case "support":
		respondText(s, i, "https://github.com/MetroReviews/support", false)
	case "claim":
		openReviewModal(s, i, types.ActionClaim, "")
	case "unclaim":
		openReviewModal(s, i, types.ActionUnclaim, "")
	case "approve":
		openReviewModal(s, i, types.ActionApprove, "")
	case "deny":
		openReviewModal(s, i, types.ActionDeny, "")
	}
}
