package api

import (
	"context"
	"errors"
	"net/http"

	"github.com/MetroReviews/backend-v2/helpers"
	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/google/uuid"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
)

func AuthList(ctx context.Context, listID uuid.UUID, key string) *uapi.HttpResponse {
	var secretKey string
	var listState types.ListState

	err := state.Pool.QueryRow(ctx,
		"SELECT secret_key, state FROM bot_list WHERE id = $1", listID,
	).Scan(&secretKey, &listState)

	if errors.Is(err, pgx.ErrNoRows) {
		resp := helpers.ErrorResponse(http.StatusNotFound, "List not found")
		return &resp
	}
	if err != nil {
		resp := helpers.ErrorResponse(http.StatusInternalServerError, "Failed to look up list: "+err.Error())
		return &resp
	}

	if key != secretKey {
		resp := helpers.ErrorResponse(http.StatusUnauthorized, "Invalid secret key")
		return &resp
	}

	if !types.IsGoodListState(listState) {
		resp := helpers.ErrorResponse(http.StatusUnauthorized, "List blacklisted, defunct or in an unknown state")
		return &resp
	}

	return nil
}
