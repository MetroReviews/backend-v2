// Package team exposes GET /team: the review team, sourced from the
// roles/user_roles tables (see the roles/perms packages) rather than a
// live Discord guild lookup. Those tables are already kept in sync with
// Discord role membership by roles.SyncMember/SyncGuild, so reading them
// directly here means /team works without hitting the Discord API (or
// even needing the bot connected) on every request.
package team

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/MetroReviews/backend-v2/helpers"
	"github.com/MetroReviews/backend-v2/perms"
	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/bwmarrin/discordgo"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
)

const tagName = "Team"

type Router struct{}

func (Router) Tag() (string, string) {
	return tagName, "The Metro Reviews review team."
}

func (Router) Routes(r *chi.Mux) {
	uapi.Route{
		Method:  uapi.GET,
		Pattern: "/team",
		OpId:    "our_team",
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Our Team",
				Description: "Returns the review team: every user holding a role that grants the queue.review permission (see the roles/perms packages). Read from the database, not a live Discord lookup.",
				Resp:        []types.TeamMember{},
				RespName:    "TeamMemberArray",
			}
		},
		Handler: ourTeam,
	}.Route(r)
}

// memberRoles accumulates one user's held role names and effective
// permission set while scanning ourTeam's query, one row per role held.
type memberRoles struct {
	discordID   *int64
	username    *string
	avatar      *string
	roleNames   []string
	permissions []string
}

func ourTeam(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	rows, err := state.Pool.Query(d.Context, `
		SELECT u.id, u.username, u.avatar, da.discord_id, r.name, r.permissions
		FROM user_roles ur
		JOIN users u ON u.id = ur.user_id
		JOIN roles r ON r.id = ur.role_id
		LEFT JOIN discord_accounts da ON da.user_id = u.id
		ORDER BY u.username`)
	if err != nil {
		return helpers.InternalError(err)
	}
	defer rows.Close()

	members := map[uuid.UUID]*memberRoles{}
	var order []uuid.UUID
	for rows.Next() {
		var userID uuid.UUID
		var username, avatar *string
		var discordID *int64
		var roleName string
		var rolePermissions []string
		if err := rows.Scan(&userID, &username, &avatar, &discordID, &roleName, &rolePermissions); err != nil {
			return helpers.InternalError(err)
		}

		m, ok := members[userID]
		if !ok {
			m = &memberRoles{discordID: discordID, username: username, avatar: avatar}
			members[userID] = m
			order = append(order, userID)
		}
		m.roleNames = append(m.roleNames, roleName)
		m.permissions = perms.Union(m.permissions, rolePermissions)
	}
	if err := rows.Err(); err != nil {
		return helpers.InternalError(err)
	}

	team := []types.TeamMember{}
	for _, userID := range order {
		m := members[userID]
		if !perms.Has(m.permissions, perms.QueueReview) {
			continue
		}
		// TeamMember is inherently Discord-shaped (ID/Avatar are used to
		// @mention and show the member) — nothing sensible to show for an
		// account with no linked Discord, so skip it rather than return a
		// broken-looking entry.
		if m.discordID == nil {
			continue
		}

		var listRoles []string
		var isListOwner bool
		for _, name := range m.roleNames {
			lower := strings.ToLower(name)
			if strings.Contains(lower, "list") && !strings.HasPrefix(lower, "list") {
				listRoles = append(listRoles, name)
			}
			if strings.EqualFold(name, "list owner") {
				isListOwner = true
			}
		}

		discordIDStr := strconv.FormatInt(*m.discordID, 10)
		username := ""
		if m.username != nil {
			username = *m.username
		}
		var avatarHash string
		if m.avatar != nil {
			avatarHash = *m.avatar
		}
		// AvatarURL is pure string-building (no network call) — safe to
		// call on a struct built from our own DB columns instead of a
		// live discordgo.User fetched from the API.
		discordUser := &discordgo.User{ID: discordIDStr, Username: username, Discriminator: "0", Avatar: avatarHash}

		team = append(team, types.TeamMember{
			Username:    username,
			ID:          discordIDStr,
			Avatar:      discordUser.AvatarURL(""),
			IsListOwner: isListOwner,
			Sudo:        perms.Has(m.permissions, perms.Wildcard),
			Roles:       listRoles,
		})
	}

	return uapi.HttpResponse{Json: team}
}
