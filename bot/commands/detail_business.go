package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

// buildBusinessDetailEmbed is buildBotDetailEmbed's counterpart for businesses.
func buildBusinessDetailEmbed(businessID uuid.UUID) (*discordgo.MessageEmbed, types.State, error) {
	var (
		name, categorySlug                  string
		description, website, address, city *string
		country                             *string
		submittedByUsername                 *string
		submittedByDiscordID                *int64
		reviewerID                          *uuid.UUID
		reviewerUsername                    *string
		reviewerDiscordID                   *int64
		st                                  types.State
		createdAt                           time.Time
	)

	err := state.Pool.QueryRow(state.Context, `
		SELECT l.name, c.slug, l.description, l.website, l.address, l.city, l.country,
		       su.username, sda.discord_id,
		       l.reviewer, ru.username, rda.discord_id,
		       l.status, l.created_at
		FROM businesses l
		JOIN categories c ON c.id = l.category_id
		JOIN users su ON su.id = l.submitted_by
		LEFT JOIN discord_accounts sda ON sda.user_id = su.id
		LEFT JOIN users ru ON ru.id = l.reviewer
		LEFT JOIN discord_accounts rda ON rda.user_id = l.reviewer
		WHERE l.id = $1`, businessID).Scan(
		&name, &categorySlug, &description, &website, &address, &city, &country,
		&submittedByUsername, &submittedByDiscordID,
		&reviewerID, &reviewerUsername, &reviewerDiscordID,
		&st, &createdAt,
	)
	if err != nil {
		return nil, 0, err
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s %s", stateEmoji[st], name),
		Description: derefOr(description, "No description provided."),
		Color:       0x5865f2,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Status", Value: stateNames[st], Inline: true},
			{Name: "Business ID", Value: businessID.String(), Inline: true},
			{Name: "Category", Value: categorySlug, Inline: true},
			{Name: "Submitted By", Value: mentionOrName(submittedByDiscordID, submittedByUsername), Inline: true},
		},
		Footer: &discordgo.MessageEmbedFooter{Text: "Submitted " + createdAt.Format("2006-01-02 15:04 MST")},
	}

	if reviewerID != nil {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "Reviewer", Value: mentionOrName(reviewerDiscordID, reviewerUsername), Inline: true})
	}
	if website != nil && *website != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "Website", Value: *website, Inline: true})
	}

	location := strings.TrimSpace(strings.Join(nonEmpty(address, city, country), ", "))
	if location != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "Location", Value: location, Inline: true})
	}

	return embed, st, nil
}
