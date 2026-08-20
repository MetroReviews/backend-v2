package rpc

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/MetroReviews/backend-v2/helpers"
	"github.com/MetroReviews/backend-v2/perms"
	"github.com/MetroReviews/backend-v2/review"
	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func ReviewBusiness(ctx context.Context, actor Actor, businessID uuid.UUID, action types.Action, reason string) (review.Result, error) {
	if err := authorize(ctx, actor, perms.QueueReview); err != nil {
		return review.Result{}, err
	}
	res := review.ApplyBusinessAction(ctx, businessID, action, reason, actor.UserID)
	announceReviewAction(actor, "business", businessID.String(), action, reason, res)
	return res, nil
}

func ReviewProject(ctx context.Context, actor Actor, projectID uuid.UUID, action types.Action, reason string) (review.Result, error) {
	if err := authorize(ctx, actor, perms.QueueReview); err != nil {
		return review.Result{}, err
	}
	res := review.ApplyProjectAction(ctx, projectID, action, reason, actor.UserID)
	announceReviewAction(actor, "project", projectID.String(), action, reason, res)
	return res, nil
}

var reviewActionNames = map[types.Action]string{
	types.ActionClaim:   "Claim",
	types.ActionUnclaim: "Unclaim",
	types.ActionApprove: "Approve",
	types.ActionDeny:    "Deny",
}

var reviewSubjectLabels = map[string]string{"business": "Business", "project": "Project"}

func announceReviewAction(actor Actor, subjectType, id string, action types.Action, reason string, res review.Result) {
	if state.Discord == nil || state.Config.LogsChannelID() == 0 {
		return
	}

	name := reviewActionNames[action]
	if name == "" {
		name = "Unknown"
	}

	fields := []*discordgo.MessageEmbedField{
		{Name: reviewSubjectLabels[subjectType], Value: fmt.Sprintf("`%s`", id), Inline: true},
		{Name: "Reviewer", Value: mentionActor(actor), Inline: true},
		{Name: "Via", Value: actor.Source, Inline: true},
	}
	if reason != "" {
		fields = append(fields, &discordgo.MessageEmbedField{Name: "Reason", Value: helpers.Truncate(reason, 1024)})
	}

	title, color := "✅ "+res.Message, 0x57f287
	if !res.OK {
		title, color = "❌ "+res.Message, 0xed4245
	}
	fields = append(fields, &discordgo.MessageEmbedField{Name: "Result", Value: title})

	embed := &discordgo.MessageEmbed{
		Title:     fmt.Sprintf("📋 %s", name),
		Color:     color,
		Fields:    fields,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	channelID := strconv.FormatUint(state.Config.LogsChannelID(), 10)
	if _, err := state.Discord.ChannelMessageSendEmbed(channelID, embed); err != nil {
		state.Logger.Error("[rpc] failed to send review action log", zap.Error(err))
	}
}
