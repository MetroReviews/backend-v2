// This file (plus detail_bot.go and detail_business.go) renders the
// full-detail view shown by the queue's "select an item" menu — this file
// is the dispatch + shared helpers, detail_bot.go/detail_business.go each
// render one subject type.
package commands

import (
	"fmt"
	"strconv"

	"github.com/MetroReviews/backend-v2/types"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

// buildDetailEmbed dispatches to the bot, business or project detail view
// depending on which one was picked from the queue's "select an item" menu.
func buildDetailEmbed(subjectType, id string) (*discordgo.MessageEmbed, types.State, error) {
	switch subjectType {
	case "bot":
		botID, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			return nil, 0, err
		}
		return buildBotDetailEmbed(botID)
	case "business":
		businessID, err := uuid.Parse(id)
		if err != nil {
			return nil, 0, err
		}
		return buildBusinessDetailEmbed(businessID)
	case "project":
		projectID, err := uuid.Parse(id)
		if err != nil {
			return nil, 0, err
		}
		return buildProjectDetailEmbed(projectID)
	default:
		return nil, 0, fmt.Errorf("unknown subject type %q", subjectType)
	}
}

// mentionOrName formats a Metro user for display in a Discord embed: a
// proper @mention if they have a linked Discord account, else their
// username, else a plain fallback. Every account today is Discord-linked,
// but the schema no longer guarantees it.
func mentionOrName(discordID *int64, username *string) string {
	if discordID != nil {
		return fmt.Sprintf("<@%d>", *discordID)
	}
	if username != nil && *username != "" {
		return *username
	}
	return "Unknown"
}

func derefOr(s *string, fallback string) string {
	if s == nil || *s == "" {
		return fallback
	}
	return *s
}

func nonEmpty(vals ...*string) []string {
	out := make([]string, 0, len(vals))
	for _, v := range vals {
		if v != nil && *v != "" {
			out = append(out, *v)
		}
	}
	return out
}
