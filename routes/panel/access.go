package panel

import (
	"encoding/base64"
	"errors"
	"net/http"
	"strconv"

	"github.com/MetroReviews/backend-v2/state"
	"github.com/infinitybotlist/eureka/jsonimpl"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
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
		"SELECT nonce FROM users WHERE user_id = $1 AND nonce = $2", userID, ticket.Nonce,
	).Scan(&dbNonce)
	if errors.Is(err, pgx.ErrNoRows) || err != nil {
		return uapi.HttpResponse{Json: panelAccessResponse{Access: false}}
	}

	gid := strconv.FormatUint(state.Config.GuildID(), 10)
	member, err := state.Discord.GuildMember(gid, ticket.UserID)
	if err != nil || member == nil {
		return uapi.HttpResponse{Json: panelAccessResponse{Access: false}}
	}

	isReviewer := state.Config.IsOwner(userID)
	for _, roleID := range member.Roles {
		if roleID == state.Config.Reviewer {
			isReviewer = true
		}
	}

	if !isReviewer {
		return uapi.HttpResponse{Json: panelAccessResponse{Access: false}}
	}

	return uapi.HttpResponse{Json: panelAccessResponse{
		Access: true,
		Member: &panelMember{ID: member.User.ID, Name: member.User.Username},
	}}
}
