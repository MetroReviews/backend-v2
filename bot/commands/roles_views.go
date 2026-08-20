package commands

import (
	"fmt"

	"github.com/MetroReviews/backend-v2/helpers"
	"github.com/MetroReviews/backend-v2/perms"
	"github.com/MetroReviews/backend-v2/roles"
	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

func buildRolesListView() (*discordgo.MessageEmbed, []discordgo.MessageComponent, error) {
	list, err := roles.List(state.Context)
	if err != nil {
		return nil, nil, err
	}

	embed := &discordgo.MessageEmbed{
		Title: "🔑 Roles",
		Color: 0x5865f2,
	}

	if len(list) == 0 {
		embed.Description = "No roles yet — use **Create Role** below to make one."
	} else {
		embed.Footer = &discordgo.MessageEmbedFooter{Text: fmt.Sprintf("%d role(s)", len(list))}
	}

	shown := list
	if len(shown) > roleListLimit {
		shown = shown[:roleListLimit]
		embed.Footer = &discordgo.MessageEmbedFooter{Text: fmt.Sprintf("Showing %d of %d role(s)", roleListLimit, len(list))}
	}

	opts := make([]discordgo.SelectMenuOption, 0, len(shown))
	for _, role := range shown {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:  role.Name,
			Value: discordRoleValue(role) + "\n" + permissionsValue(role.Permissions),
		})
		opts = append(opts, discordgo.SelectMenuOption{
			Label:       role.Name,
			Value:       role.ID.String(),
			Description: helpers.Truncate(permissionsSummary(role.Permissions), 100),
		})
	}

	var components []discordgo.MessageComponent
	if len(opts) > 0 {
		components = append(components, row(discordgo.SelectMenu{
			CustomID:    "role:select",
			Placeholder: "🔑 Select a role to manage…",
			Options:     opts,
		}))
	}
	components = append(components, discordgo.ActionsRow{Components: []discordgo.MessageComponent{
		discordgo.Button{Label: "➕ Create Role", Style: discordgo.SuccessButton, CustomID: "role:create"},
		discordgo.Button{Label: "🔄 Sync from Discord", Style: discordgo.SecondaryButton, CustomID: "role:syncguild"},
	}})

	return embed, components, nil
}

func buildRoleDetailView(roleID uuid.UUID) (*discordgo.MessageEmbed, []discordgo.MessageComponent, error) {
	role, err := roles.Get(state.Context, roleID)
	if err != nil {
		return nil, nil, err
	}
	memberCount, err := roles.MemberCount(state.Context, roleID)
	if err != nil {
		return nil, nil, err
	}

	embed := &discordgo.MessageEmbed{
		Title: "🔑 " + role.Name,
		Color: 0x5865f2,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Discord Role", Value: discordRoleValue(*role)},
			{Name: "Permissions", Value: permissionsValue(role.Permissions)},
			{Name: "Members", Value: fmt.Sprintf("%d", memberCount)},
		},
	}

	id := role.ID.String()
	buttons := []discordgo.MessageComponent{
		discordgo.Button{Label: "Rename", Style: discordgo.SecondaryButton, CustomID: "role:rename:" + id},
		discordgo.Button{Label: "Edit Permissions", Style: discordgo.SecondaryButton, CustomID: "role:permsedit:" + id},
		discordgo.Button{Label: "Link Discord Role", Style: discordgo.SecondaryButton, CustomID: "role:linkstart:" + id},
	}
	if role.DiscordRoleID != nil {
		buttons = append(buttons, discordgo.Button{Label: "Unlink", Style: discordgo.SecondaryButton, CustomID: "role:unlink:" + id})
	}
	buttons = append(buttons, discordgo.Button{Label: "Delete", Style: discordgo.DangerButton, CustomID: "role:delete:" + id})

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: buttons},
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{Label: "◀ Back to Roles", Style: discordgo.SecondaryButton, CustomID: "role:backlist"},
		}},
	}

	return embed, components, nil
}

func backButton(roleID string) discordgo.MessageComponent {
	return discordgo.ActionsRow{Components: []discordgo.MessageComponent{
		discordgo.Button{Label: "◀ Back", Style: discordgo.SecondaryButton, CustomID: "role:back:" + roleID},
	}}
}

func buildPermissionEditView(role *types.Role) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	embed := &discordgo.MessageEmbed{
		Title:       "🔑 " + role.Name + " — Edit Permissions",
		Description: "Select every permission this role should grant, then submit. This replaces the role's entire permission set — reselect the ones you want to keep.",
		Color:       0x5865f2,
	}

	granted := func(slug string) bool { return perms.Has(role.Permissions, slug) }

	opts := []discordgo.SelectMenuOption{
		{
			Label:       "Everything (Sudo)",
			Value:       perms.Wildcard,
			Description: "Grants every permission, including future ones",
			Default:     granted(perms.Wildcard),
		},
	}
	for _, p := range perms.Catalog {
		opts = append(opts, discordgo.SelectMenuOption{
			Label:       p.Slug,
			Value:       p.Slug,
			Description: helpers.Truncate(p.Description, 100),
			Default:     granted(p.Slug),
		})
	}

	components := []discordgo.MessageComponent{
		row(discordgo.SelectMenu{
			CustomID:    "role:permset:" + role.ID.String(),
			Placeholder: "Select permissions…",
			MinValues:   ptrInt(0),
			MaxValues:   len(opts),
			Options:     opts,
		}),
		backButton(role.ID.String()),
	}

	return embed, components
}

func buildLinkRolePickerView(role *types.Role) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	embed := &discordgo.MessageEmbed{
		Title:       "🔑 " + role.Name + " — Link Discord Role",
		Description: "Pick the Discord server role whose membership should drive this role. Whoever holds that Discord role will automatically get this role's permissions, kept in sync by the bot.",
		Color:       0x5865f2,
	}

	components := []discordgo.MessageComponent{
		row(discordgo.SelectMenu{
			MenuType:    discordgo.RoleSelectMenu,
			CustomID:    "role:linkset:" + role.ID.String(),
			Placeholder: "Select a Discord role…",
			MaxValues:   1,
		}),
		backButton(role.ID.String()),
	}

	return embed, components
}

func buildDeleteConfirmView(role *types.Role, memberCount int) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	embed := &discordgo.MessageEmbed{
		Title:       "⚠️ Delete " + role.Name + "?",
		Description: fmt.Sprintf("This permanently deletes the role and revokes it from %d member(s). This can't be undone.", memberCount),
		Color:       0xed4245,
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{Label: "⚠️ Confirm Delete", Style: discordgo.DangerButton, CustomID: "role:deleteconfirm:" + role.ID.String()},
			discordgo.Button{Label: "Cancel", Style: discordgo.SecondaryButton, CustomID: "role:back:" + role.ID.String()},
		}},
	}

	return embed, components
}
