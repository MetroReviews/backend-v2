package commands

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/MetroReviews/backend-v2/helpers"
	"github.com/MetroReviews/backend-v2/silverpelt"
	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

// actionNames gives the human label for a review action, used in both the
// modal title and the logs-channel entry.
var actionNames = map[types.Action]string{
	types.ActionClaim:   "Claim",
	types.ActionUnclaim: "Unclaim",
	types.ActionApprove: "Approve",
	types.ActionDeny:    "Deny",
}

// resultStatus summarizes a silverpelt.Response into per-list status fields
// plus an overall title/color. Shared by the reviewer-facing result embed
// and the logs-channel entry so both stay in sync.
func resultStatus(res *silverpelt.Response) (fields []*discordgo.MessageEmbedField, title string, color int) {
	if res.Message != "" {
		return nil, "❌ Request Failed", 0xed4245 // red
	}

	names := make([]string, 0, len(res.Lists))
	for name := range res.Lists {
		names = append(names, name)
	}
	sort.Strings(names)

	allOK := true
	fields = make([]*discordgo.MessageEmbedField, 0, len(names))
	for _, name := range names {
		resp := res.Lists[name]
		ok := resp.Status >= 200 && resp.Status < 300
		allOK = allOK && ok

		emoji := "✅"
		if !ok {
			emoji = "❌"
		}
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   name,
			Value:  fmt.Sprintf("%s `%d %s`", emoji, resp.Status, http.StatusText(resp.Status)),
			Inline: true,
		})
	}

	title, color = "✅ Request Sent", 0x57f287 // green
	switch {
	case len(fields) == 0:
		title, color = "❌ No Lists Notified", 0xed4245 // red
	case !allOK:
		title, color = "⚠️ Request Sent", 0xfee75c // yellow
	}
	return fields, title, color
}

// buildResultEmbed turns a silverpelt.Response into a clean status embed.
// No raw error text ever goes to Discord: a top-level failure just shows
// the (already human-safe) message, and per-list results show a check or
// cross with the HTTP status code and its standard status text.
func buildResultEmbed(res *silverpelt.Response) *discordgo.MessageEmbed {
	fields, title, color := resultStatus(res)
	embed := &discordgo.MessageEmbed{Title: title, Color: color, Fields: fields}
	if res.Message != "" {
		embed.Description = res.Message
	}
	return embed
}

// logAction posts a record of a claim/unclaim/approve/deny to the
// configured logs channel, reusing the same per-list status summary shown
// to the reviewer. No-op if logs_channel isn't configured.
func logAction(s *discordgo.Session, action types.Action, botID, reviewerID int64, reason string, resend bool, res *silverpelt.Response) {
	channelID := state.Config.LogsChannelID()
	if channelID == 0 {
		return
	}

	name := actionNames[action]
	if name == "" {
		name = "Unknown"
	}

	fields := []*discordgo.MessageEmbedField{
		{Name: "Bot", Value: fmt.Sprintf("`%d`", botID), Inline: true},
		{Name: "Reviewer", Value: fmt.Sprintf("<@%d>", reviewerID), Inline: true},
	}
	if resend {
		fields = append(fields, &discordgo.MessageEmbedField{Name: "Resend", Value: "Yes", Inline: true})
	}
	if action != types.ActionClaim && reason != "" {
		fields = append(fields, &discordgo.MessageEmbedField{Name: "Reason", Value: helpers.Truncate(reason, 1024)})
	}

	listFields, _, color := resultStatus(res)
	fields = append(fields, listFields...)
	if res.Message != "" {
		fields = append(fields, &discordgo.MessageEmbedField{Name: "Error", Value: res.Message})
	}

	embed := &discordgo.MessageEmbed{
		Title:     fmt.Sprintf("📋 %s", name),
		Color:     color,
		Fields:    fields,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	if _, err := s.ChannelMessageSendEmbed(strconv.FormatUint(channelID, 10), embed); err != nil {
		state.Logger.Error("[bot] failed to send action log", zap.Error(err))
	}
}
