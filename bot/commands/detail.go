package commands

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/MetroReviews/backend-v2/helpers"
	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/bwmarrin/discordgo"
)

// buildBotDetailEmbed renders the full-detail view shown by the "View bot
// details" select menu — everything the compact list rows don't have room for.
func buildBotDetailEmbed(botID int64) (*discordgo.MessageEmbed, types.State, error) {
	var (
		username, description, longDescription    string
		website, invite, support, library, prefix *string
		reviewNote                                *string
		owner                                     int64
		reviewer                                  *int64
		tags                                      []string
		nsfw                                      bool
		st                                        types.State
		addedAt                                   time.Time
	)

	err := state.Pool.QueryRow(state.Context, `
		SELECT username, description, long_description, website, invite,
		       support, library, prefix, review_note, owner, reviewer,
		       tags, nsfw, state, added_at
		FROM bot_queue WHERE bot_id = $1`, botID).Scan(
		&username, &description, &longDescription, &website, &invite,
		&support, &library, &prefix, &reviewNote, &owner, &reviewer,
		&tags, &nsfw, &st, &addedAt,
	)
	if err != nil {
		return nil, 0, err
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s %s", stateEmoji[st], username),
		Description: description,
		Color:       0x5865f2,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Status", Value: stateNames[st], Inline: true},
			{Name: "Bot ID", Value: strconv.FormatInt(botID, 10), Inline: true},
			{Name: "Owner", Value: fmt.Sprintf("<@%d>", owner), Inline: true},
		},
		Footer: &discordgo.MessageEmbedFooter{Text: "Added " + addedAt.Format("2006-01-02 15:04 MST")},
	}

	if reviewer != nil {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "Reviewer", Value: fmt.Sprintf("<@%d>", *reviewer), Inline: true})
	}
	if prefix != nil && *prefix != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "Prefix", Value: *prefix, Inline: true})
	}
	if nsfw {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "NSFW", Value: "Yes", Inline: true})
	}
	if len(tags) > 0 {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "Tags", Value: strings.Join(tags, ", ")})
	}

	inviteVal := helpers.InviteURL(strconv.FormatInt(botID, 10))
	if invite != nil && *invite != "" {
		inviteVal = *invite
	}
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "Invite", Value: inviteVal})

	if website != nil && *website != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "Website", Value: *website, Inline: true})
	}
	if support != nil && *support != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "Support", Value: *support, Inline: true})
	}
	if library != nil && *library != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "Library", Value: *library, Inline: true})
	}
	if reviewNote != nil && *reviewNote != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "Review Note", Value: helpers.Truncate(*reviewNote, 1024)})
	}
	if longDescription != "" && longDescription != description {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "Long Description", Value: helpers.Truncate(longDescription, 1024)})
	}

	return embed, st, nil
}
