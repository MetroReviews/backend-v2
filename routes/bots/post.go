package bots

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/MetroReviews/backend-v2/api"
	"github.com/MetroReviews/backend-v2/bot"
	"github.com/MetroReviews/backend-v2/helpers"
	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

func postBot(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	if _, resp := api.AuthUser(d.Context, r); resp != nil {
		return *resp
	}

	var payload types.BotPost
	if hresp, ok := uapi.MarshalReq(r, &payload); !ok {
		return hresp
	}

	rem := []string{}

	botID, err := strconv.ParseInt(payload.BotID, 10, 64)
	if err != nil {
		return helpers.ErrorResponse(http.StatusBadRequest, "Invalid bot fields")
	}
	owner, err := strconv.ParseInt(payload.Owner, 10, 64)
	if err != nil {
		return helpers.ErrorResponse(http.StatusBadRequest, "Invalid bot fields")
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

	extraOwners = helpers.DedupeExcluding(extraOwners, owner)

	var banner, website, support *string
	if payload.Banner != nil {
		if v, ok := helpers.HTTPSify(*payload.Banner); ok {
			banner = &v
		} else {
			rem = append(rem, "banner")
		}
	}
	if payload.Website != nil {
		if v, ok := helpers.HTTPSify(*payload.Website); ok {
			website = &v
		} else {
			rem = append(rem, "website")
		}
	}
	if payload.Support != nil {
		if v, ok := helpers.HTTPSify(*payload.Support); ok {
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
		if !helpers.Contains(tags, "utility") {
			tags = append(tags, "utility")
		}
	} else {
		tags = []string{"utility"}
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
		return helpers.ErrorResponse(http.StatusBadRequest, "Bot does not exist?")
	}

	// INSERT ... ON CONFLICT DO NOTHING RETURNING replaces a separate
	// SELECT-then-INSERT existence check: that pair raced with concurrent
	// submissions of the same bot (both could pass the SELECT before either
	// INSERT lands), where this is a single atomic statement.
	var inserted int64
	err = state.Pool.QueryRow(d.Context, `
		INSERT INTO bots (
			bot_id, username, banner, description, long_description,
			website, invite, owner, support, donate, library, nsfw, prefix, tags,
			review_note, extra_owners, state
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17
		)
		ON CONFLICT (bot_id) DO NOTHING
		RETURNING bot_id`,
		botID, user.Username, banner, payload.Description, payload.LongDescription,
		website, invite, owner, support, payload.Donate, payload.Library, payload.NSFW, payload.Prefix, tags,
		payload.ReviewNote, extraOwners, types.StatePending,
	).Scan(&inserted)
	if errors.Is(err, pgx.ErrNoRows) {
		return helpers.ErrorResponse(http.StatusConflict, "Bot already in queue")
	}
	if err != nil {
		return helpers.InternalError(err)
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
