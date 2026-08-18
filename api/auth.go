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

// AuthUser authenticates a logged-in user from the "Authorization: Bearer
// <session_token>" header minted on login (see routes/panel/callback.go).
// Returns the authenticated user, or a ready-to-return HttpResponse
// explaining why authentication failed.
func AuthUser(ctx context.Context, r *http.Request) (*types.User, *uapi.HttpResponse) {
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if token == "" {
		resp := helpers.ErrorResponse(http.StatusUnauthorized, "Login required")
		return nil, &resp
	}

	var u types.User
	err := state.Pool.QueryRow(ctx, `
		SELECT u.id, da.discord_id, u.username, u.avatar, u.bio, u.is_staff, u.banned
		FROM discord_accounts da JOIN users u ON u.id = da.user_id
		WHERE da.session_token = $1 AND da.session_expires_at > NOW()`, token,
	).Scan(&u.ID, &u.DiscordID, &u.Username, &u.Avatar, &u.Bio, &u.IsStaff, &u.Banned)
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

// AuthStaff is AuthUser plus a staff-membership check, for endpoints that
// moderate bots/businesses/reviews (review actions, resolving reports/claims).
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

// AuthPermission is AuthUser plus a check that the user holds perm (see the
// perms/roles packages), for panel endpoints gated more finely than plain
// staff membership — managing roles, banning users, and similar.
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

// isConfigOwner reports whether u is a config.yaml owner — always
// permitted regardless of what roles/permissions (if any) they hold, same
// as the bot-side checks in bot/commands/util.go and the panel access
// check in routes/panel/access.go.
func isConfigOwner(u *types.User) bool {
	return u.DiscordID != nil && state.Config.IsOwner(*u.DiscordID)
}
