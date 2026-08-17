package bots

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/MetroReviews/backend-v2/api"
	"github.com/MetroReviews/backend-v2/helpers"
	"github.com/MetroReviews/backend-v2/silverpelt"
	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
)

func actionBot(d uapi.RouteData, r *http.Request, action types.Action) uapi.HttpResponse {
	botID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		return uapi.DefaultResponse(http.StatusBadRequest)
	}

	reviewer, err := strconv.ParseInt(r.URL.Query().Get("reviewer"), 10, 64)
	if err != nil {
		return helpers.ErrorResponse(http.StatusBadRequest, "Invalid reviewer")
	}

	listID, err := uuid.Parse(r.URL.Query().Get("list_id"))
	if err != nil {
		return uapi.DefaultResponse(http.StatusBadRequest)
	}

	if resp := api.AuthList(d.Context, listID, r.Header.Get("Authorization")); resp != nil {
		return *resp
	}

	var reason types.Reason
	if hresp, ok := uapi.MarshalReq(r, &reason); !ok {
		return hresp
	}

	res := silverpelt.Handle(d.Context, silverpelt.Request{
		BotID:    botID,
		Reason:   reason.Reason,
		Resend:   true,
		Action:   action,
		Reviewer: reviewer,
	})

	return htmlResponse(res.ToHTML())
}

func reapproveBot(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	botID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		return uapi.DefaultResponse(http.StatusBadRequest)
	}

	var botState types.State
	err = state.Pool.QueryRow(d.Context, "SELECT state FROM bot_queue WHERE bot_id = $1", botID).Scan(&botState)
	if errors.Is(err, pgx.ErrNoRows) || botState != types.StateApproved {
		return htmlResponse("Bot is not approved and cannot be reapproved!")
	}
	if err != nil {
		return helpers.InternalError(err)
	}

	reviewer := int64(0)
	if state.Discord != nil && state.Discord.State != nil && state.Discord.State.User != nil {
		reviewer, _ = strconv.ParseInt(state.Discord.State.User.ID, 10, 64)
	}

	res := silverpelt.Handle(d.Context, silverpelt.Request{
		BotID:    botID,
		Reason:   "Already approved, readding due to errors (Automated Action)",
		Resend:   true,
		Action:   types.ActionApprove,
		Reviewer: reviewer,
	})

	return htmlResponse(res.ToHTML())
}

func htmlResponse(body string) uapi.HttpResponse {
	return uapi.HttpResponse{
		Data:    body,
		Headers: map[string]string{"Content-Type": "text/html; charset=utf-8"},
	}
}
