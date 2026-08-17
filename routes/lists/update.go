package lists

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/MetroReviews/backend-v2/api"
	"github.com/MetroReviews/backend-v2/helpers"
	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/infinitybotlist/eureka/crypto"
	"github.com/infinitybotlist/eureka/uapi"
)

func updateList(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	listID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return uapi.DefaultResponse(http.StatusBadRequest)
	}

	if resp := api.AuthList(d.Context, listID, r.Header.Get("Authorization")); resp != nil {
		return *resp
	}

	var update types.ListUpdate
	if hresp, ok := uapi.MarshalReq(r, &update); !ok {
		return hresp
	}

	// Column names below are all literal strings chosen by this code, never
	// derived from request input, so building the SET clause with fmt is safe.
	var setClauses []string
	var args []any
	hasUpdated := []string{}

	set := func(column string, value any, label string) {
		args = append(args, value)
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", column, len(args)))
		hasUpdated = append(hasUpdated, label)
	}

	if update.Name != nil && *update.Name != "" {
		set("name", *update.Name, "name")
	}
	if update.Description != nil && *update.Description != "" {
		set("description", *update.Description, "description")
	}
	if update.Icon != nil && strings.HasPrefix(*update.Icon, "https://") {
		set("icon", *update.Icon, "icon")
	}
	if update.ClaimBotAPI != nil && *update.ClaimBotAPI != "" {
		set("claim_bot_api", *update.ClaimBotAPI, "claim_bot_api")
	}
	if update.UnclaimBotAPI != nil && *update.UnclaimBotAPI != "" {
		set("unclaim_bot_api", *update.UnclaimBotAPI, "unclaim_bot_api")
	}
	if update.ApproveBotAPI != nil && *update.ApproveBotAPI != "" {
		set("approve_bot_api", *update.ApproveBotAPI, "approve_bot_api")
	}
	if update.DenyBotAPI != nil && *update.DenyBotAPI != "" {
		set("deny_bot_api", *update.DenyBotAPI, "deny_bot_api")
	}
	if update.Domain != nil && *update.Domain != "" {
		set("domain", strings.TrimSuffix(*update.Domain, "/"), "domain")
	}

	if len(hasUpdated) > 0 && update.ResetSecretKey {
		return helpers.ErrorResponse(http.StatusBadRequest, "Cannot reset secret key while updating other fields")
	}

	if update.ResetSecretKey {
		key := crypto.RandString(43)
		if _, err := state.Pool.Exec(d.Context,
			"UPDATE bot_list SET secret_key = $1 WHERE id = $2", key, listID); err != nil {
			return helpers.InternalError(err)
		}
		return uapi.HttpResponse{Json: types.SecretKeyResponse{SecretKey: key}}
	}

	if len(setClauses) > 0 {
		args = append(args, listID)
		query := fmt.Sprintf("UPDATE bot_list SET %s WHERE id = $%d", strings.Join(setClauses, ", "), len(args))
		if _, err := state.Pool.Exec(d.Context, query, args...); err != nil {
			return helpers.InternalError(err)
		}
	}

	return uapi.HttpResponse{Json: types.UpdatedResponse{HasUpdated: hasUpdated}}
}
