package api

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/MetroReviews/backend-v2/helpers"
	"github.com/MetroReviews/backend-v2/roles"
	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
)

func AuthUser(ctx context.Context, r *http.Request) (*types.User, *uapi.HttpResponse) {
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if token == "" {
		resp := helpers.ErrorResponse(http.StatusUnauthorized, "Login required")
		return nil, &resp
	}

	var u types.User
	err := state.Pool.QueryRow(ctx, `
		SELECT u.id, da.discord_id, u.username, u.avatar, u.bio, u.is_staff, u.banned, u.created_at
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		LEFT JOIN discord_accounts da ON da.user_id = u.id
		WHERE s.token = $1 AND s.expires_at > NOW()`, token,
	).Scan(&u.ID, &u.DiscordID, &u.Username, &u.Avatar, &u.Bio, &u.IsStaff, &u.Banned, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		resp := helpers.ErrorResponse(http.StatusUnauthorized, "Invalid or expired session")
		return nil, &resp
	}
	if err != nil {
		resp := helpers.InternalError(err)
		return nil, &resp
	}

	if u.Banned {
		resp := helpers.ErrorResponse(http.StatusForbidden, "This account is banned")
		return nil, &resp
	}

	return &u, nil
}

func AuthStaff(ctx context.Context, r *http.Request) (*types.User, *uapi.HttpResponse) {
	u, resp := AuthUser(ctx, r)
	if resp != nil {
		return nil, resp
	}
	if !u.IsStaff && !isConfigOwner(u) {
		errResp := helpers.ErrorResponse(http.StatusForbidden, "Staff only")
		return nil, &errResp
	}
	return u, nil
}

func AuthPermission(ctx context.Context, r *http.Request, perm string) (*types.User, *uapi.HttpResponse) {
	u, resp := AuthUser(ctx, r)
	if resp != nil {
		return nil, resp
	}
	if isConfigOwner(u) {
		return u, nil
	}

	has, err := roles.HasPermission(ctx, u.ID, perm)
	if err != nil {
		errResp := helpers.InternalError(err)
		return nil, &errResp
	}
	if !has {
		errResp := helpers.ErrorResponse(http.StatusForbidden, "Missing required permission: "+perm)
		return nil, &errResp
	}
	return u, nil
}

func isConfigOwner(u *types.User) bool {
	return u.DiscordID != nil && state.Config.IsOwner(*u.DiscordID)
}
