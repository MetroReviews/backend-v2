package commands

import (
	"fmt"

	"github.com/MetroReviews/backend-v2/types"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

func buildDetailEmbed(subjectType, id string) (*discordgo.MessageEmbed, types.State, error) {
	switch subjectType {
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
