package commands

import (
	"fmt"
	"strconv"
	"time"

	"github.com/MetroReviews/backend-v2/helpers"
	"github.com/MetroReviews/backend-v2/review"
	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

var actionNames = map[types.Action]string{
	types.ActionClaim:   "Claim",
	types.ActionUnclaim: "Unclaim",
	types.ActionApprove: "Approve",
	types.ActionDeny:    "Deny",
}

func resultTitle(res review.Result) (title string, color int) {
	if !res.OK {
		return "❌ " + res.Message, 0xed4245
	}
	return "✅ " + res.Message, 0x57f287
}

func buildResultEmbed(res review.Result) *discordgo.MessageEmbed {
	title, color := resultTitle(res)
	return &discordgo.MessageEmbed{Title: title, Color: color}
}

func logAction(s *discordgo.Session, action types.Action, subjectType, id string, reviewerID int64, reason string, res review.Result) {
	channelID := state.Config.LogsChannelID()
	if channelID == 0 {
		return
	}

	name := actionNames[action]
	if name == "" {
		name = "Unknown"
	}

	fields := []*discordgo.MessageEmbedField{
		{Name: subjectLabel[subjectType], Value: fmt.Sprintf("`%s`", id), Inline: true},
		{Name: "Reviewer", Value: fmt.Sprintf("<@%d>", reviewerID), Inline: true},
	}
	if reason != "" {
		fields = append(fields, &discordgo.MessageEmbedField{Name: "Reason", Value: helpers.Truncate(reason, 1024)})
	}

	title, color := resultTitle(res)
	fields = append(fields, &discordgo.MessageEmbedField{Name: "Result", Value: title})

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
