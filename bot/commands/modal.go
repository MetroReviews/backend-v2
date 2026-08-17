package commands

import (
	"strconv"
	"strings"

	"github.com/MetroReviews/backend-v2/silverpelt"
	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

func modalID(action types.Action) string {
	switch action {
	case types.ActionClaim:
		return "claim_modal"
	case types.ActionUnclaim:
		return "unclaim_modal"
	case types.ActionApprove:
		return "approve_modal"
	case types.ActionDeny:
		return "deny_modal"
	}
	return "unknown_modal"
}

func actionFromModalID(id string) (types.Action, bool) {
	switch id {
	case "claim_modal":
		return types.ActionClaim, true
	case "unclaim_modal":
		return types.ActionUnclaim, true
	case "approve_modal":
		return types.ActionApprove, true
	case "deny_modal":
		return types.ActionDeny, true
	}
	return 0, false
}

// openReviewModal opens the claim/unclaim/approve/deny modal. presetBotID
// pre-fills the Bot ID field (still editable) when opened from a /queue
// action button, so the reviewer isn't retyping an ID they just saw; pass ""
// for the plain /claim /unclaim /approve /deny slash commands.
func openReviewModal(s *discordgo.Session, i *discordgo.InteractionCreate, action types.Action, presetBotID string) {
	title := map[types.Action]string{
		types.ActionClaim:   "Claim Bot",
		types.ActionUnclaim: "Unclaim Bot",
		types.ActionApprove: "Approve Bot",
		types.ActionDeny:    "Deny Bot",
	}[action]

	components := []discordgo.MessageComponent{
		row(&discordgo.TextInput{
			CustomID: "bot_id", Label: "Bot ID", Style: discordgo.TextInputShort, Required: true,
			Value: presetBotID,
		}),
	}

	if action != types.ActionClaim {
		components = append(components, row(&discordgo.TextInput{
			CustomID: "reason", Label: "Reason", Style: discordgo.TextInputParagraph,
			Required: true, MinLength: 5, MaxLength: 4000,
		}))
	}

	components = append(components, row(&discordgo.TextInput{
		CustomID: "resend", Label: "Resend to other lists (owner only, T/F)",
		Style: discordgo.TextInputShort, Required: false, Value: "F",
	}))

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID:   modalID(action),
			Title:      title,
			Components: components,
		},
	})
	if err != nil {
		state.Logger.Error("[bot] failed to open modal", zap.Error(err))
	}
}

// HandleModal handles the claim/unclaim/approve/deny modal submissions
// opened by openReviewModal.
func HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ModalSubmitData()
	action, ok := actionFromModalID(data.CustomID)
	if !ok {
		return
	}

	if !isReviewer(i) {
		respondText(s, i, "You are not a reviewer", true)
		return
	}

	values := modalValues(data)

	botID, err := strconv.ParseInt(strings.TrimSpace(values["bot_id"]), 10, 64)
	if err != nil {
		respondText(s, i, "Bot ID invalid", false)
		return
	}

	resend := false
	switch strings.ToLower(strings.TrimSpace(values["resend"])) {
	case "t", "true":
		resend = true
	}

	userID := interactionUserID(i)

	if resend && !state.Config.IsOwner(userID) {
		respondText(s, i, "You are not an owner", false)
		return
	}

	reason := "STUB_REASON"
	if action != types.ActionClaim {
		reason = values["reason"]
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	}); err != nil {
		state.Logger.Error("[bot] failed to defer modal response", zap.Error(err))
		return
	}

	res := silverpelt.Handle(state.Context, silverpelt.Request{
		BotID:    botID,
		Reason:   reason,
		Resend:   resend,
		Action:   action,
		Reviewer: userID,
	})

	embed := buildResultEmbed(res)
	if _, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{embed},
	}); err != nil {
		state.Logger.Error("[bot] failed to send review result", zap.Error(err))
	}

	logAction(s, action, botID, userID, reason, resend, res)
}
