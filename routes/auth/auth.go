package auth

import (
	"net/http"
	"strconv"
	"time"

	"github.com/MetroReviews/backend-v2/helpers"
	"github.com/MetroReviews/backend-v2/identity"
	"github.com/MetroReviews/backend-v2/roles"
	"github.com/MetroReviews/backend-v2/state"
	"github.com/go-chi/chi/v5"
	"github.com/infinitybotlist/eureka/jsonimpl"
	"github.com/infinitybotlist/eureka/uapi"
)

const tagName = "Auth"

type Router struct{}

func (Router) Tag() (string, string) {
	return tagName, "Login: Discord access-token exchange, email/password register & login, and linking a password onto an existing account."
}

func (Router) Routes(r *chi.Mux) {
	registerRoutes(r)
}

type loginRequest struct {
	AccessToken string `json:"access_token" validate:"required" msg:"A Discord access token is required" description:"A Discord OAuth2 access token the caller already obtained (identify scope)"`
	TokenType   string `json:"token_type" description:"The Discord token type; defaults to Bearer"`
}

type loginResponse struct {
	SessionToken string    `json:"session_token" description:"The Metro session token to send as 'Authorization: Bearer <token>'"`
	ExpiresAt    time.Time `json:"expires_at" description:"When the session token expires"`
	UserID       string    `json:"user_id" description:"The authenticated user's Discord ID"`
	Username     string    `json:"username" description:"The authenticated user's Discord username"`
	Avatar       string    `json:"avatar" description:"The authenticated user's Discord avatar hash"`
}

func login(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	if resp := helpers.RateLimit(r, "auth-login", 20, time.Hour); resp != nil {
		return *resp
	}

	var payload loginRequest
	if hresp, ok := uapi.MarshalReq(r, &payload); !ok {
		return hresp
	}

	tokenType := payload.TokenType
	if tokenType == "" {
		tokenType = "Bearer"
	}

	req, _ := http.NewRequestWithContext(d.Context, http.MethodGet, "https://discord.com/api/v10/users/@me", nil)
	req.Header.Set("Authorization", tokenType+" "+payload.AccessToken)
	userResp, err := http.DefaultClient.Do(req)
	if err != nil {
		return helpers.ErrorResponse(http.StatusBadGateway, err.Error())
	}
	defer userResp.Body.Close()

	var user struct {
		ID       string `json:"id"`
		Username string `json:"username"`
		Avatar   string `json:"avatar"`
	}
	if err := jsonimpl.UnmarshalReader(userResp.Body, &user); err != nil || userResp.StatusCode != http.StatusOK || user.ID == "" {
		return helpers.ErrorResponse(http.StatusUnauthorized, "Invalid or expired Discord access token")
	}

	discordID, err := strconv.ParseInt(user.ID, 10, 64)
	if err != nil {
		return helpers.ErrorResponse(http.StatusBadRequest, "Malformed Discord user ID")
	}

	metroUserID, err := identity.EnsureDiscordUser(d.Context, discordID, user.Username)
	if err != nil {
		return helpers.InternalError(err)
	}

	if _, err := state.Pool.Exec(d.Context,
		"UPDATE users SET username = $1, avatar = $2 WHERE id = $3",
		user.Username, user.Avatar, metroUserID,
	); err != nil {
		return helpers.InternalError(err)
	}

	var memberRoles []string
	if state.Discord != nil {
		gid := strconv.FormatUint(state.Config.GuildID(), 10)
		if member, mErr := state.Discord.GuildMember(gid, user.ID); mErr == nil && member != nil {
			memberRoles = member.Roles
		}
	}
	if err := roles.SyncMember(d.Context, metroUserID, discordID, memberRoles); err != nil {
		return helpers.InternalError(err)
	}

	sessionToken, expiresAt, err := identity.NewSession(d.Context, metroUserID)
	if err != nil {
		return helpers.InternalError(err)
	}

	return uapi.HttpResponse{Json: loginResponse{
		SessionToken: sessionToken,
		ExpiresAt:    expiresAt,
		UserID:       user.ID,
		Username:     user.Username,
		Avatar:       user.Avatar,
	}}
}
