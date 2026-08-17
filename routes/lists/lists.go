package lists

import (
	"errors"
	"net/http"
	"strings"

	"github.com/MetroReviews/backend-v2/api"
	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/infinitybotlist/eureka/crypto"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
)

const tagName = "Lists"

type Router struct{}

func (Router) Tag() (string, string) {
	return tagName, "Bot lists enrolled in Metro Reviews."
}

func (Router) Routes(r *chi.Mux) {
	uapi.Route{
		Method:  uapi.GET,
		Pattern: "/list/{id}",
		OpId:    "get_list",
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Get List",
				Description: "Gets a single bot list by its ID.",
				Resp:        types.List{},
				Params: []docs.Parameter{
					{Name: "id", In: "path", Description: "The list's ID", Required: true, Schema: docs.IdSchema},
				},
			}
		},
		Handler: getList,
	}.Route(r)

	uapi.Route{
		Method:  uapi.GET,
		Pattern: "/lists",
		OpId:    "get_all_lists",
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Get All Lists",
				Description: "Gets every bot list enrolled in Metro Reviews.",
				Resp:        []types.List{},
				RespName:    "ListArray",
			}
		},
		Handler: getAllLists,
	}.Route(r)

	uapi.Route{
		Method:  uapi.PATCH,
		Pattern: "/lists/{id}",
		OpId:    "update_list",
		Auth:    []uapi.AuthType{{Type: "List"}},
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Update List",
				Description: "Updates a list's fields. Requires the list's secret key in the Authorization header. If `reset_secret_key` is true it must be the only change requested.",
				Req:         types.ListUpdate{},
				Resp:        types.UpdatedResponse{},
				Params: []docs.Parameter{
					{Name: "id", In: "path", Description: "The list's ID", Required: true, Schema: docs.IdSchema},
				},
			}
		},
		Handler: updateList,
	}.Route(r)
}

func getList(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return uapi.DefaultResponse(http.StatusBadRequest)
	}

	rows, err := state.Pool.Query(d.Context,
		"SELECT id, name, description, domain, state, icon FROM bot_list WHERE id = $1", id)
	if err != nil {
		return uapi.HttpResponse{Status: http.StatusInternalServerError, Json: types.ApiError{Message: err.Error(), Error: true}}
	}

	list, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[types.List])
	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.DefaultResponse(http.StatusNotFound)
	}
	if err != nil {
		return uapi.HttpResponse{Status: http.StatusInternalServerError, Json: types.ApiError{Message: err.Error(), Error: true}}
	}

	return uapi.HttpResponse{Json: list}
}

func getAllLists(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	rows, err := state.Pool.Query(d.Context,
		"SELECT id, name, description, domain, state, icon FROM bot_list ORDER BY id ASC")
	if err != nil {
		return uapi.HttpResponse{Status: http.StatusInternalServerError, Json: types.ApiError{Message: err.Error(), Error: true}}
	}

	lists, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.List])
	if err != nil {
		return uapi.HttpResponse{Status: http.StatusInternalServerError, Json: types.ApiError{Message: err.Error(), Error: true}}
	}

	return uapi.HttpResponse{Json: lists}
}

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

	hasUpdated := []string{}

	set := func(column string, value any, label string) uapi.HttpResponse {
		if _, err := state.Pool.Exec(d.Context,
			"UPDATE bot_list SET "+column+" = $1 WHERE id = $2", value, listID); err != nil {
			return uapi.HttpResponse{Status: http.StatusInternalServerError, Json: types.ApiError{Message: err.Error(), Error: true}}
		}
		hasUpdated = append(hasUpdated, label)
		return uapi.HttpResponse{}
	}

	if update.Name != nil && *update.Name != "" {
		if resp := set("name", *update.Name, "name"); resp.Status != 0 {
			return resp
		}
	}
	if update.Description != nil && *update.Description != "" {
		if resp := set("description", *update.Description, "description"); resp.Status != 0 {
			return resp
		}
	}
	if update.Icon != nil && strings.HasPrefix(*update.Icon, "https://") {
		if resp := set("icon", *update.Icon, "icon"); resp.Status != 0 {
			return resp
		}
	}
	if update.ClaimBotAPI != nil && *update.ClaimBotAPI != "" {
		if resp := set("claim_bot_api", *update.ClaimBotAPI, "claim_bot_api"); resp.Status != 0 {
			return resp
		}
	}
	if update.UnclaimBotAPI != nil && *update.UnclaimBotAPI != "" {
		if resp := set("unclaim_bot_api", *update.UnclaimBotAPI, "unclaim_bot_api"); resp.Status != 0 {
			return resp
		}
	}
	if update.ApproveBotAPI != nil && *update.ApproveBotAPI != "" {
		if resp := set("approve_bot_api", *update.ApproveBotAPI, "approve_bot_api"); resp.Status != 0 {
			return resp
		}
	}
	if update.DenyBotAPI != nil && *update.DenyBotAPI != "" {
		if resp := set("deny_bot_api", *update.DenyBotAPI, "deny_bot_api"); resp.Status != 0 {
			return resp
		}
	}
	if update.Domain != nil && *update.Domain != "" {
		domain := strings.TrimSuffix(*update.Domain, "/")
		if resp := set("domain", domain, "domain"); resp.Status != 0 {
			return resp
		}
	}

	if len(hasUpdated) > 0 && update.ResetSecretKey {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Cannot reset secret key while updating other fields", Error: true},
		}
	} else if update.ResetSecretKey {
		key := crypto.RandString(43)
		if _, err := state.Pool.Exec(d.Context,
			"UPDATE bot_list SET secret_key = $1 WHERE id = $2", key, listID); err != nil {
			return uapi.HttpResponse{Status: http.StatusInternalServerError, Json: types.ApiError{Message: err.Error(), Error: true}}
		}
		return uapi.HttpResponse{Json: types.SecretKeyResponse{SecretKey: key}}
	}

	return uapi.HttpResponse{Json: types.UpdatedResponse{HasUpdated: hasUpdated}}
}
