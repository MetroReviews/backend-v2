package commands

import (
	"strings"

	"github.com/MetroReviews/backend-v2/roles"
	"github.com/MetroReviews/backend-v2/state"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	roleCreateModalID     = "role_create_modal"
	roleRenameModalPrefix = "role_rename_modal:"
	roleNameModalFieldID  = "name"
	roleNameMaxLength     = 100
)

func nameField(prefill string) discordgo.MessageComponent {
	return row(&discordgo.TextInput{
		CustomID: roleNameModalFieldID, Label: "Role name", Style: discordgo.TextInputShort,
		Required: true, MaxLength: roleNameMaxLength, Value: prefill,
	})
}

func openRoleCreateModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID:   roleCreateModalID,
			Title:      "Create Role",
			Components: []discordgo.MessageComponent{nameField("")},
		},
	})
	if err != nil {
		state.Logger.Error("[bot] failed to open role create modal", zap.Error(err))
	}
}

func openRoleRenameModal(s *discordgo.Session, i *discordgo.InteractionCreate, roleID uuid.UUID) {
	role, err := roles.Get(state.Context, roleID)
	if err != nil {
		state.Logger.Error("[bot] failed to load role for rename", zap.Error(err))
		return
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID:   roleRenameModalPrefix + roleID.String(),
			Title:      "Rename Role",
			Components: []discordgo.MessageComponent{nameField(role.Name)},
		},
	}); err != nil {
		state.Logger.Error("[bot] failed to open role rename modal", zap.Error(err))
	}
}

// handleRoleModal handles the create/rename modal submissions opened
// above, editing the /roles message the triggering button lived on in
// place — the same "one message, redrawn in place" flow every other
// /roles transition uses.
func handleRoleModal(s *discordgo.Session, i *discordgo.InteractionCreate, data discordgo.ModalSubmitInteractionData) {
	if !isRoleManager(i) {
		respondText(s, i, "You don't have permission to manage roles.", true)
		return
	}

	name := strings.TrimSpace(modalValues(data)[roleNameModalFieldID])
	if name == "" {
		respondText(s, i, "A name is required.", true)
		return
	}

	if !deferUpdate(s, i) {
		return
	}

	var roleID uuid.UUID
	switch {
	case data.CustomID == roleCreateModalID:
		role, err := roles.Create(state.Context, name, nil, []string{})
		if err != nil {
			if isUniqueViolation(err) {
				followupError(s, i, "A role named \""+name+"\" already exists.")
			} else {
				state.Logger.Error("[bot] failed to create role", zap.Error(err))
			}
			return
		}
		roleID = role.ID

	case strings.HasPrefix(data.CustomID, roleRenameModalPrefix):
		id, err := uuid.Parse(strings.TrimPrefix(data.CustomID, roleRenameModalPrefix))
		if err != nil {
			return
		}
		if _, err := roles.Update(state.Context, id, &name, nil, false, nil); err != nil {
			if isUniqueViolation(err) {
				followupError(s, i, "A role named \""+name+"\" already exists.")
			} else {
				state.Logger.Error("[bot] failed to rename role", zap.Error(err))
			}
			return
		}
		roleID = id

	default:
		return
	}

	embed, components, err := buildRoleDetailView(roleID)
	if err != nil {
		state.Logger.Error("[bot] failed to load role after modal submit", zap.Error(err))
		return
	}
	if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
	}); err != nil {
		state.Logger.Error("[bot] failed to edit role view after modal submit", zap.Error(err))
	}
}
