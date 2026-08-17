package api

import (
	"context"
	"errors"
	"net/http"

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
		return &uapi.HttpResponse{
			Status: http.StatusNotFound,
			Json:   types.ApiError{Message: "List not found", Error: true},
		}
	}
	if err != nil {
		return &uapi.HttpResponse{
			Status: http.StatusInternalServerError,
			Json:   types.ApiError{Message: "Failed to look up list: " + err.Error(), Error: true},
		}
	}

	if key != secretKey {
		return &uapi.HttpResponse{
			Status: http.StatusUnauthorized,
			Json:   types.ApiError{Message: "Invalid secret key", Error: true},
		}
	}

	if !types.IsGoodListState(listState) {
		return &uapi.HttpResponse{
			Status: http.StatusUnauthorized,
			Json:   types.ApiError{Message: "List blacklisted, defunct or in an unknown state", Error: true},
		}
	}

	return nil
}
