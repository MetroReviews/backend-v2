package commands

import (
	"strings"

	"github.com/MetroReviews/backend-v2/identity"
	"github.com/MetroReviews/backend-v2/review"
	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func modalID(action types.Action, subjectType string) string {
	word := map[types.Action]string{
		types.ActionClaim:   "claim",
		types.ActionUnclaim: "unclaim",
		types.ActionApprove: "approve",
		types.ActionDeny:    "deny",
	}[action]
	if word == "" {
		word = "unknown"
	}
	return word + "_modal:" + subjectType
}

func actionFromModalID(id string) (action types.Action, subjectType string, ok bool) {
	word, subjectType, found := strings.Cut(id, ":")
	if !found {
		return 0, "", false
	}
	switch word {
	case "claim_modal":
		return types.ActionClaim, subjectType, true
	case "unclaim_modal":
		return types.ActionUnclaim, subjectType, true
	case "approve_modal":
		return types.ActionApprove, subjectType, true
	case "deny_modal":
		return types.ActionDeny, subjectType, true
	}
	return 0, "", false
}

func openReviewModal(s *discordgo.Session, i *discordgo.InteractionCreate, subjectType string, action types.Action, presetID string) {
	titleWord := map[types.Action]string{
		types.ActionClaim:   "Claim",
		types.ActionUnclaim: "Unclaim",
		types.ActionApprove: "Approve",
		types.ActionDeny:    "Deny",
	}[action]
	title := titleWord + " " + subjectLabel[subjectType]

	components := []discordgo.MessageComponent{
		row(&discordgo.TextInput{
			CustomID: "id", Label: subjectLabel[subjectType] + " ID", Style: discordgo.TextInputShort, Required: true,
			Value: presetID,
		}),
		row(&discordgo.TextInput{
			CustomID: "reason", Label: "Reason", Style: discordgo.TextInputParagraph,
			Required: true, MinLength: 5, MaxLength: 4000,
		}),
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID:   modalID(action, subjectType),
			Title:      title,
			Components: components,
		},
	})
	if err != nil {
		state.Logger.Error("[bot] failed to open modal", zap.Error(err))
	}
}

func HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ModalSubmitData()

	if data.CustomID == roleCreateModalID || strings.HasPrefix(data.CustomID, roleRenameModalPrefix) {
		handleRoleModal(s, i, data)
		return
	}

	action, subjectType, ok := actionFromModalID(data.CustomID)
	if !ok {
		return
	}

	if !isReviewer(i) {
		respondText(s, i, "You are not a reviewer", true)
		return
	}

	values := modalValues(data)
	id := strings.TrimSpace(values["id"])
	reason := values["reason"]
	userID := interactionUserID(i)

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	}); err != nil {
		state.Logger.Error("[bot] failed to defer modal response", zap.Error(err))
		return
	}

	reviewerID, err := identity.EnsureDiscordUser(state.Context, userID, interactionUsername(i))
	if err != nil {
		state.Logger.Error("[bot] failed to resolve reviewer identity", zap.Error(err))
		content := "Failed to resolve your Metro account. Try again in a moment."
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content})
		return
	}

	var res review.Result
	switch subjectType {
	case "business":
		businessID, err := uuid.Parse(id)
		if err != nil {
			res = review.Result{Message: "Business ID invalid"}
		} else {
			res = review.ApplyBusinessAction(state.Context, businessID, action, reason, reviewerID)
		}
	case "project":
		projectID, err := uuid.Parse(id)
		if err != nil {
			res = review.Result{Message: "Project ID invalid"}
		} else {
			res = review.ApplyProjectAction(state.Context, projectID, action, reason, reviewerID)
		}
	default:
		res = review.Result{Message: "Unknown subject type"}
	}

	embed := buildResultEmbed(res)
	if _, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{embed},
	}); err != nil {
		state.Logger.Error("[bot] failed to send review result", zap.Error(err))
	}

	logAction(s, action, subjectType, id, userID, reason, res)
}
