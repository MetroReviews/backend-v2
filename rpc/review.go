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

// ReviewBot claims/unclaims/approves/denies botID on actor's behalf: the
// single entry point routes/bots/action.go (HTTP) and
// bot/commands/modal.go (Discord) both call, enforcing queue.review and
// posting the outcome to the logs channel regardless of which surface
// triggered it.
func ReviewBot(ctx context.Context, actor Actor, botID int64, action types.Action, reason string) (review.Result, error) {
	if err := authorize(ctx, actor, perms.QueueReview); err != nil {
		return review.Result{}, err
	}
	res := review.ApplyBotAction(ctx, botID, action, reason, actor.UserID)
	announceReviewAction(actor, "bot", strconv.FormatInt(botID, 10), action, reason, res)
	return res, nil
}

// ReviewBusiness is ReviewBot's counterpart for businesses.
func ReviewBusiness(ctx context.Context, actor Actor, businessID uuid.UUID, action types.Action, reason string) (review.Result, error) {
	if err := authorize(ctx, actor, perms.QueueReview); err != nil {
		return review.Result{}, err
	}
	res := review.ApplyBusinessAction(ctx, businessID, action, reason, actor.UserID)
	announceReviewAction(actor, "business", businessID.String(), action, reason, res)
	return res, nil
}

// ReviewProject is ReviewBot's counterpart for projects — a business's
// posted portfolio items, which go through the same queue.
func ReviewProject(ctx context.Context, actor Actor, projectID uuid.UUID, action types.Action, reason string) (review.Result, error) {
	if err := authorize(ctx, actor, perms.QueueReview); err != nil {
		return review.Result{}, err
	}
	res := review.ApplyProjectAction(ctx, projectID, action, reason, actor.UserID)
	announceReviewAction(actor, "project", projectID.String(), action, reason, res)
	return res, nil
}

// reviewActionNames and subjectLabels give the human labels an audit embed
// needs — small, local lookup tables, same style as the maps
// bot/commands/queue.go and modal.go keep for their own embeds.
var reviewActionNames = map[types.Action]string{
	types.ActionClaim:   "Claim",
	types.ActionUnclaim: "Unclaim",
	types.ActionApprove: "Approve",
	types.ActionDeny:    "Deny",
}

var reviewSubjectLabels = map[string]string{"bot": "Bot", "business": "Business", "project": "Project"}

// announceReviewAction posts a record of a claim/unclaim/approve/deny to
// the configured logs channel — a no-op if Discord isn't connected or no
// logs channel is configured. Runs off state.Discord directly rather than
// a caller-supplied session, so it fires the same way whether the action
// came from the API or from Discord.
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

	title, color := "✅ "+res.Message, 0x57f287 // green
	if !res.OK {
		title, color = "❌ "+res.Message, 0xed4245 // red
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
