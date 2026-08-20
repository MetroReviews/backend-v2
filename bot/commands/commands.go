package commands

import (
	"strconv"

	"github.com/MetroReviews/backend-v2/types"
	"github.com/bwmarrin/discordgo"
)

var reviewTypeOption = &discordgo.ApplicationCommandOption{
	Type:        discordgo.ApplicationCommandOptionString,
	Name:        "type",
	Description: "What you're reviewing",
	Required:    true,
	Choices: []*discordgo.ApplicationCommandOptionChoice{
		{Name: "Business", Value: "business"},
		{Name: "Project", Value: "project"},
	},
}

var Definitions = []*discordgo.ApplicationCommand{
	{
		Name:        "queue",
		Description: "Show the review queue (businesses and projects)",
		Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionBoolean, Name: "show_all", Description: "Show all states", Required: false},
		},
	},
	{Name: "syncroles", Description: "Re-syncs panel roles/permissions from this server's Discord roles"},
	{Name: "roles", Description: "Create, edit and delete panel roles"},
	{Name: "support", Description: "Show the link to support"},
	{Name: "claim", Description: "Claim a business or project for review", Options: []*discordgo.ApplicationCommandOption{reviewTypeOption}},
	{Name: "unclaim", Description: "Unclaim a business or project", Options: []*discordgo.ApplicationCommandOption{reviewTypeOption}},
	{Name: "approve", Description: "Approve a business or project", Options: []*discordgo.ApplicationCommandOption{reviewTypeOption}},
	{Name: "deny", Description: "Deny a business or project", Options: []*discordgo.ApplicationCommandOption{reviewTypeOption}},
}

func Sync(s *discordgo.Session, guildID uint64) error {
	gid := strconv.FormatUint(guildID, 10)
	_, err := s.ApplicationCommandBulkOverwrite(s.State.User.ID, gid, Definitions)
	return err
}

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

func reviewType(data discordgo.ApplicationCommandInteractionData) string {
	for _, opt := range data.Options {
		if opt.Name == "type" {
			return opt.StringValue()
		}
	}
	return "business"
}
