package panel

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"strconv"

	"github.com/MetroReviews/backend-v2/helpers"
	"github.com/MetroReviews/backend-v2/perms"
	"github.com/MetroReviews/backend-v2/roles"
	"github.com/MetroReviews/backend-v2/state"
	"github.com/infinitybotlist/eureka/jsonimpl"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

func getPanelAccess(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	raw := r.URL.Query().Get("ticket")

	decoded, err := base64.URLEncoding.DecodeString(raw)
	if err != nil {
		return uapi.HttpResponse{Json: panelAccessResponse{Access: false}}
	}

	var ticket struct {
		Nonce  string `json:"nonce"`
		UserID string `json:"user_id"`
	}
	if err := jsonimpl.Unmarshal(decoded, &ticket); err != nil {
		return uapi.HttpResponse{Json: panelAccessResponse{Access: false}}
	}

	if ticket.Nonce == "" || ticket.UserID == "" {
		return uapi.HttpResponse{Json: panelAccessResponse{Access: false}}
	}

	if !checkNonceTime(ticket.Nonce) {
		return uapi.HttpResponse{Json: panelAccessResponse{Access: false, Hint: "Nonce expiry"}}
	}

	userID, err := strconv.ParseInt(ticket.UserID, 10, 64)
	if err != nil {
		return uapi.HttpResponse{Json: panelAccessResponse{Access: false}}
	}

	var dbNonce string
	err = state.Pool.QueryRow(d.Context,
		"SELECT nonce FROM discord_accounts WHERE discord_id = $1 AND nonce = $2", userID, ticket.Nonce,
	).Scan(&dbNonce)
	if errors.Is(err, pgx.ErrNoRows) || err != nil {
		return uapi.HttpResponse{Json: panelAccessResponse{Access: false}}
	}

	gid := strconv.FormatUint(state.Config.GuildID(), 10)
	member, err := state.Discord.GuildMember(gid, ticket.UserID)
	if err != nil || member == nil {
		return uapi.HttpResponse{Json: panelAccessResponse{Access: false}}
	}

	if !isStaffMember(d.Context, ticket.UserID) {
		return uapi.HttpResponse{Json: panelAccessResponse{Access: false}}
	}

	return uapi.HttpResponse{Json: panelAccessResponse{
		Access: true,
		Member: &panelMember{ID: member.User.ID, Name: member.User.Username},
	}}
}

// isStaffMember reports whether the given Discord user ID is a configured
// owner or currently holds a Discord role that grants the panel.access
// permission (see the perms/roles packages) in the main guild — Metro's
// definition of "staff" for both the staff panel gate and the users.is_staff
// column minted on login. It re-checks live against Discord rather than
// trusting the DB's possibly-stale is_staff, since this runs at the moment
// panel access is actually granted.
func isStaffMember(ctx context.Context, userID string) bool {
	id, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return false
	}
	if state.Config.IsOwner(id) {
		return true
	}

	if state.Discord == nil {
		return false
	}

	staffRoleIDs, err := roles.DiscordRoleIDsWithPermission(ctx, perms.PanelAccess)
	if err != nil {
		state.Logger.Error("[panel] failed to load staff role IDs", zap.Error(err))
		return false
	}
	if len(staffRoleIDs) == 0 {
		return false
	}

	gid := strconv.FormatUint(state.Config.GuildID(), 10)
	member, err := state.Discord.GuildMember(gid, userID)
	if err != nil || member == nil {
		return false
	}
	for _, roleID := range member.Roles {
		rid, err := strconv.ParseInt(roleID, 10, 64)
		if err != nil {
			continue
		}
		if helpers.Contains(staffRoleIDs, rid) {
			return true
		}
	}
	return false
}
