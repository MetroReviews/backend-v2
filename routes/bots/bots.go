package bots

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/MetroReviews/backend-v2/api"
	"github.com/MetroReviews/backend-v2/bot"
	"github.com/MetroReviews/backend-v2/silverpelt"
	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

const tagName = "Bots"

const botColumns = `bot_id, username, banner, description, long_description, website, invite, owner, extra_owners, support, donate, library, nsfw, prefix, tags, review_note, cross_add, state, list_source, added_at, reviewer, invite_link`

type Router struct{}

func (Router) Tag() (string, string) {
	return tagName, "Bots in the review queue."
}

func (Router) Routes(r *chi.Mux) {
	uapi.Route{
		Method:  uapi.GET,
		Pattern: "/bots/{id}",
		OpId:    "get_bot",
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Get Bot",
				Description: "Gets a single queued bot by its Discord ID.",
				Resp:        types.Bot{},
				Params: []docs.Parameter{
					{Name: "id", In: "path", Description: "The bot's Discord ID", Required: true, Schema: docs.IdSchema},
				},
			}
		},
		Handler: getBot,
	}.Route(r)

	uapi.Route{
		Method:  uapi.GET,
		Pattern: "/bots",
		OpId:    "get_all_bots",
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Get All Bots",
				Description: "Gets every bot in the review queue.",
				Resp:        []types.Bot{},
				RespName:    "BotArray",
			}
		},
		Handler: getAllBots,
	}.Route(r)

	uapi.Route{
		Method:  uapi.POST,
		Pattern: "/bots",
		OpId:    "post_bots",
		Auth:    []uapi.AuthType{{Type: "List"}},
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Add Bot",
				Description: "Adds a bot to the review queue. Requires the list secret key in the Authorization header and the `list_id` query parameter. All optional fields are genuinely optional.",
				Req:         types.BotPost{},
				Resp:        types.PostBotResponse{},
				Params: []docs.Parameter{
					{Name: "list_id", In: "query", Description: "The submitting list's ID", Required: true, Schema: docs.IdSchema},
				},
			}
		},
		Handler: postBot,
	}.Route(r)

	uapi.Route{
		Method:  uapi.POST,
		Pattern: "/bots/{id}/approve",
		OpId:    "approve_bot",
		Auth:    []uapi.AuthType{{Type: "List"}},
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Approve Bot",
				Description: "Approves a bot and propagates the action to all enrolled lists. Requires the list secret key and `list_id`/`reviewer` query parameters.",
				Req:         types.Reason{},
				Resp:        types.ApiError{},
				Params: []docs.Parameter{
					{Name: "id", In: "path", Description: "The bot's Discord ID", Required: true, Schema: docs.IdSchema},
					{Name: "reviewer", In: "query", Description: "The reviewer's Discord ID", Required: true, Schema: docs.IdSchema},
					{Name: "list_id", In: "query", Description: "The list's ID", Required: true, Schema: docs.IdSchema},
				},
			}
		},
		Handler: func(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
			return actionBot(d, r, types.ActionApprove)
		},
	}.Route(r)

	uapi.Route{
		Method:  uapi.POST,
		Pattern: "/bots/{id}/deny",
		OpId:    "deny_bot",
		Auth:    []uapi.AuthType{{Type: "List"}},
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Deny Bot",
				Description: "Denies a bot and propagates the action to all enrolled lists. Requires the list secret key and `list_id`/`reviewer` query parameters.",
				Req:         types.Reason{},
				Resp:        types.ApiError{},
				Params: []docs.Parameter{
					{Name: "id", In: "path", Description: "The bot's Discord ID", Required: true, Schema: docs.IdSchema},
					{Name: "reviewer", In: "query", Description: "The reviewer's Discord ID", Required: true, Schema: docs.IdSchema},
					{Name: "list_id", In: "query", Description: "The list's ID", Required: true, Schema: docs.IdSchema},
				},
			}
		},
		Handler: func(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
			return actionBot(d, r, types.ActionDeny)
		},
	}.Route(r)

	uapi.Route{
		Method:  uapi.GET,
		Pattern: "/littlecloud/{id}",
		OpId:    "reapprove_bot",
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Reapprove Bot",
				Description: "Re-propagates an already-approved bot to all lists (used to recover from list-side errors).",
				Resp:        types.ApiError{},
				Params: []docs.Parameter{
					{Name: "id", In: "path", Description: "The bot's Discord ID", Required: true, Schema: docs.IdSchema},
				},
			}
		},
		Handler: reapproveBot,
	}.Route(r)
}

func getBot(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		return uapi.DefaultResponse(http.StatusBadRequest)
	}

	rows, err := state.Pool.Query(d.Context, "SELECT "+botColumns+" FROM bot_queue WHERE bot_id = $1", id)
	if err != nil {
		return uapi.HttpResponse{Status: http.StatusInternalServerError, Json: types.ApiError{Message: err.Error(), Error: true}}
	}

	b, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[types.Bot])
	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.DefaultResponse(http.StatusNotFound)
	}
	if err != nil {
		return uapi.HttpResponse{Status: http.StatusInternalServerError, Json: types.ApiError{Message: err.Error(), Error: true}}
	}

	return uapi.HttpResponse{Json: b}
}

func getAllBots(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	rows, err := state.Pool.Query(d.Context, "SELECT "+botColumns+" FROM bot_queue ORDER BY bot_id ASC")
	if err != nil {
		return uapi.HttpResponse{Status: http.StatusInternalServerError, Json: types.ApiError{Message: err.Error(), Error: true}}
	}

	list, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.Bot])
	if err != nil {
		return uapi.HttpResponse{Status: http.StatusInternalServerError, Json: types.ApiError{Message: err.Error(), Error: true}}
	}

	return uapi.HttpResponse{Json: list}
}

func httpsify(v string) (string, bool) {
	if strings.HasPrefix(v, "https://") {
		return v, true
	}
	v = strings.Replace(v, "http://", "https://", 1)
	if strings.HasPrefix(v, "https://") {
		return v, true
	}
	return "", false
}

func postBot(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	listID, err := uuid.Parse(r.URL.Query().Get("list_id"))
	if err != nil {
		return uapi.DefaultResponse(http.StatusBadRequest)
	}

	if resp := api.AuthList(d.Context, listID, r.Header.Get("Authorization")); resp != nil {
		return *resp
	}

	var payload types.BotPost
	if hresp, ok := uapi.MarshalReq(r, &payload); !ok {
		return hresp
	}

	rem := []string{}

	botID, err := strconv.ParseInt(payload.BotID, 10, 64)
	if err != nil {
		return uapi.HttpResponse{Status: http.StatusBadRequest, Json: types.ApiError{Message: "Invalid bot fields", Error: true}}
	}
	owner, err := strconv.ParseInt(payload.Owner, 10, 64)
	if err != nil {
		return uapi.HttpResponse{Status: http.StatusBadRequest, Json: types.ApiError{Message: "Invalid bot fields", Error: true}}
	}

	extraOwners := []int64{}
	badExtra := false
	for _, v := range payload.ExtraOwners {
		eo, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			badExtra = true
			break
		}
		extraOwners = append(extraOwners, eo)
	}
	if badExtra {
		extraOwners = []int64{}
		rem = append(rem, "extra_owners")
	}

	extraOwners = dedupeExcluding(extraOwners, owner)

	var banner, website, support *string
	if payload.Banner != nil {
		if v, ok := httpsify(*payload.Banner); ok {
			banner = &v
		} else {
			rem = append(rem, "banner")
		}
	}
	if payload.Website != nil {
		if v, ok := httpsify(*payload.Website); ok {
			website = &v
		} else {
			rem = append(rem, "website")
		}
	}
	if payload.Support != nil {
		if v, ok := httpsify(*payload.Support); ok {
			support = &v
		} else {
			rem = append(rem, "support")
		}
	}

	tags := payload.Tags
	if len(tags) > 0 {
		for i := range tags {
			tags[i] = strings.ToLower(tags[i])
		}
		if !contains(tags, "utility") {
			tags = append(tags, "utility")
		}
	} else {
		tags = []string{"utility"}
	}

	var exists int64
	err = state.Pool.QueryRow(d.Context, "SELECT bot_id FROM bot_queue WHERE bot_id = $1", botID).Scan(&exists)
	if err == nil {
		return uapi.HttpResponse{Status: http.StatusConflict, Json: types.ApiError{Message: "Bot already in queue", Error: true}}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return uapi.HttpResponse{Status: http.StatusInternalServerError, Json: types.ApiError{Message: err.Error(), Error: true}}
	}

	var invite *string
	if payload.Invite != nil {
		if strings.HasPrefix(*payload.Invite, "https://") {
			invite = payload.Invite
		} else {
			rem = append(rem, "invite")
		}
	}

	user, err := state.Discord.User(strconv.FormatInt(botID, 10))
	if err != nil || user == nil {
		return uapi.HttpResponse{Status: http.StatusBadRequest, Json: types.ApiError{Message: "Bot does not exist?", Error: true}}
	}

	crossAdd := true
	if payload.CrossAdd != nil {
		crossAdd = *payload.CrossAdd
	}

	_, err = state.Pool.Exec(d.Context, `
		INSERT INTO bot_queue (
			bot_id, username, banner, list_source, description, long_description,
			website, invite, owner, support, donate, library, nsfw, prefix, tags,
			review_note, extra_owners, cross_add, state
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19
		)`,
		botID, user.Username, banner, listID, payload.Description, payload.LongDescription,
		website, invite, owner, support, payload.Donate, payload.Library, payload.NSFW, payload.Prefix, tags,
		payload.ReviewNote, extraOwners, crossAdd, types.StatePending,
	)
	if err != nil {
		return uapi.HttpResponse{Status: http.StatusInternalServerError, Json: types.ApiError{Message: err.Error(), Error: true}}
	}

	inviteLink := ""
	if invite != nil {
		inviteLink = *invite
	}
	reviewNote := ""
	if payload.ReviewNote != nil {
		reviewNote = *payload.ReviewNote
	}
	if err := bot.AnnounceBotQueue(botID, user.Username, payload.Owner, inviteLink, reviewNote); err != nil {
		state.Logger.Error("[post_bots] failed to announce bot", zap.Error(err))
	}

	return uapi.HttpResponse{Json: types.PostBotResponse{Removed: rem}}
}

func actionBot(d uapi.RouteData, r *http.Request, action types.Action) uapi.HttpResponse {
	botID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		return uapi.DefaultResponse(http.StatusBadRequest)
	}

	reviewer, err := strconv.ParseInt(r.URL.Query().Get("reviewer"), 10, 64)
	if err != nil {
		return uapi.HttpResponse{Status: http.StatusBadRequest, Json: types.ApiError{Message: "Invalid reviewer", Error: true}}
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
		return uapi.HttpResponse{Status: http.StatusInternalServerError, Json: types.ApiError{Message: err.Error(), Error: true}}
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

func dedupeExcluding(in []int64, exclude int64) []int64 {
	seen := map[int64]struct{}{}
	out := []int64{}
	for _, v := range in {
		if v == exclude {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
