package team

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/MetroReviews/backend-v2/helpers"
	"github.com/MetroReviews/backend-v2/perms"
	"github.com/MetroReviews/backend-v2/roles"
	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/bwmarrin/discordgo"
	"github.com/go-chi/chi/v5"
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
				Description: "Returns the review team: every guild member holding the reviewer role.",
				Resp:        []types.TeamMember{},
				RespName:    "TeamMemberArray",
			}
		},
		Handler: ourTeam,
	}.Route(r)
}

func ourTeam(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	gid := strconv.FormatUint(state.Config.GuildID(), 10)

	guildRoles, err := state.Discord.GuildRoles(gid)
	if err != nil {
		return uapi.HttpResponse{Json: map[string]string{"detail": "Guild not found"}}
	}

	roleNames := make(map[string]string, len(guildRoles))
	for _, role := range guildRoles {
		roleNames[role.ID] = role.Name
	}

	var members []*discordgo.Member
	after := ""
	for {
		batch, err := state.Discord.GuildMembers(gid, after, 1000)
		if err != nil {
			return uapi.HttpResponse{Json: map[string]string{"detail": "Guild not found"}}
		}
		members = append(members, batch...)
		if len(batch) < 1000 {
			break
		}
		after = batch[len(batch)-1].User.ID
	}

	// Reviewer/Sudo aren't static config role IDs anymore — they're
	// whichever Discord role(s) the permissions system currently has
	// linked to queue.review/the wildcard (see the roles/perms packages).
	// DiscordRoleIDsWithPermission already folds the wildcard in, so a
	// Sudo-linked role satisfies "reviewer" here too. "List Owner" was
	// never part of that system (a display badge, not a permission), so
	// it's still matched by role name.
	reviewerRoleIDs, err := roles.DiscordRoleIDsWithPermission(d.Context, perms.QueueReview)
	if err != nil {
		return helpers.InternalError(err)
	}
	sudoRoleIDs, err := roles.DiscordRoleIDsWithPermission(d.Context, perms.Wildcard)
	if err != nil {
		return helpers.InternalError(err)
	}

	team := []types.TeamMember{}

	for _, member := range members {
		var listRoles []string
		var isListOwner, sudo, isReviewer bool

		for _, roleID := range member.Roles {
			name := roleNames[roleID]
			lower := strings.ToLower(name)
			if strings.Contains(lower, "list") && !strings.HasPrefix(lower, "list") {
				listRoles = append(listRoles, name)
			}
			if strings.EqualFold(name, "list owner") {
				isListOwner = true
			}

			rid, err := strconv.ParseInt(roleID, 10, 64)
			if err != nil {
				continue
			}
			if helpers.Contains(sudoRoleIDs, rid) {
				sudo = true
			}
			if helpers.Contains(reviewerRoleIDs, rid) {
				isReviewer = true
			}
		}

		if isReviewer {
			team = append(team, types.TeamMember{
				Username:    member.User.Username,
				ID:          member.User.ID,
				Avatar:      member.User.AvatarURL(""),
				IsListOwner: isListOwner,
				Sudo:        sudo,
				Roles:       listRoles,
			})
		}
	}

	return uapi.HttpResponse{Json: team}
}
