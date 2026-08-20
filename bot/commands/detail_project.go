package commands

import (
	"fmt"
	"time"

	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

func buildProjectDetailEmbed(projectID uuid.UUID) (*discordgo.MessageEmbed, types.State, error) {
	var (
		title, businessName  string
		description, url     *string
		completedAt          *time.Time
		submittedByUsername  *string
		submittedByDiscordID *int64
		reviewerID           *uuid.UUID
		reviewerUsername     *string
		reviewerDiscordID    *int64
		st                   types.State
		createdAt            time.Time
	)

	err := state.Pool.QueryRow(state.Context, `
		SELECT p.title, b.name, p.description, p.url, p.completed_at,
		       su.username, sda.discord_id,
		       p.reviewer, ru.username, rda.discord_id,
		       p.status, p.created_at
		FROM projects p
		JOIN businesses b ON b.id = p.business_id
		JOIN users su ON su.id = p.submitted_by
		LEFT JOIN discord_accounts sda ON sda.user_id = su.id
		LEFT JOIN users ru ON ru.id = p.reviewer
		LEFT JOIN discord_accounts rda ON rda.user_id = p.reviewer
		WHERE p.id = $1`, projectID).Scan(
		&title, &businessName, &description, &url, &completedAt,
		&submittedByUsername, &submittedByDiscordID,
		&reviewerID, &reviewerUsername, &reviewerDiscordID,
		&st, &createdAt,
	)
	if err != nil {
		return nil, 0, err
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s %s", stateEmoji[st], title),
		Description: derefOr(description, "No description provided."),
		Color:       0x5865f2,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Status", Value: stateNames[st], Inline: true},
			{Name: "Project ID", Value: projectID.String(), Inline: true},
			{Name: "Business", Value: businessName, Inline: true},
			{Name: "Submitted By", Value: mentionOrName(submittedByDiscordID, submittedByUsername), Inline: true},
		},
		Footer: &discordgo.MessageEmbedFooter{Text: "Submitted " + createdAt.Format("2006-01-02 15:04 MST")},
	}

	if reviewerID != nil {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "Reviewer", Value: mentionOrName(reviewerDiscordID, reviewerUsername), Inline: true})
	}
	if url != nil && *url != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "Link", Value: *url, Inline: true})
	}
	if completedAt != nil {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "Completed", Value: completedAt.Format("2006-01-02"), Inline: true})
	}

	return embed, st, nil
}
