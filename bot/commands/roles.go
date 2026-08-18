// This file (plus roles_format.go, roles_views.go, roles_util.go,
// roles_component.go, roles_component_actions.go and roles_modal.go)
// implements /roles: an interactive, in-Discord CRUD UI for the
// permissions system (see the roles/perms packages) built the same way
// /queue is — one ephemeral message, its embed and components replaced in
// place as the caller drills into a role, edits it, and backs out. Gated to
// the roles.manage permission (isRoleManager), same as /syncroles.
//
// roles_format.go formats role fields for display, roles_views.go builds
// each screen's embed/components, roles_util.go holds small shared
// utilities, and this file is just the /roles command handler.
package commands

import (
	"github.com/MetroReviews/backend-v2/state"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

// roleListLimit caps how many roles the picker shows — Discord's select
// menus top out at 25 options, and roles are hand-configured/few by
// nature, so this is a soft ceiling rather than real pagination.
const roleListLimit = 25

// cmdRoles handles /roles.
func cmdRoles(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !isRoleManager(i) {
		respondText(s, i, "You don't have permission to manage roles.", true)
		return
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Flags: discordgo.MessageFlagsEphemeral},
	}); err != nil {
		state.Logger.Error("[bot] failed to defer roles response", zap.Error(err))
		return
	}

	embed, components, err := buildRolesListView()
	if err != nil {
		state.Logger.Error("[bot] cmdRoles failed to load roles", zap.Error(err))
		content := "Failed to load roles."
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content}) //nolint:errcheck
		return
	}

	if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
	}); err != nil {
		state.Logger.Error("[bot] failed to edit deferred roles response", zap.Error(err))
	}
}
