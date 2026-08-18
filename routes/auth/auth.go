// Package auth exposes POST /auth/login: a token-introspection login for
// first-party clients (the public website) that have already run their own
// Discord OAuth2 handshake and hold a Discord access token, but need a
// Metro session_token to make authenticated calls (posting reviews, filing
// claims, ...).
//
// It is the non-panel sibling of routes/panel/frostpaw: frostpaw exchanges
// an OAuth code and hands the session back inside a redirect ticket for the
// staff panel, whereas this verifies a caller-supplied access token
// directly against Discord and returns the session as JSON. Both mint the
// exact same discord_accounts.session_token that api.AuthUser validates, so
// a website login is indistinguishable from a panel login to every
// downstream handler.
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
	"github.com/infinitybotlist/eureka/crypto"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/jsonimpl"
	"github.com/infinitybotlist/eureka/uapi"
)

// sessionTTL matches the panel login's session lifetime (see
// routes/panel/callback.go) — a website login is just as long-lived.
const sessionTTL = 30 * 24 * time.Hour

const tagName = "Auth"

type Router struct{}

func (Router) Tag() (string, string) {
	return tagName, "First-party login: exchange a Discord access token for a Metro session."
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

func (Router) Routes(r *chi.Mux) {
	uapi.Route{
		Method:  uapi.POST,
		Pattern: "/auth/login",
		OpId:    "auth_login",
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Login",
				Description: "Verifies a Discord access token against Discord and mints a Metro session token for it, creating the Metro user on first login. Intended for first-party clients that run their own Discord OAuth2 flow.",
				Req:         loginRequest{},
				Resp:        loginResponse{},
			}
		},
		Handler: login,
	}.Route(r)
}

func login(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var payload loginRequest
	if hresp, ok := uapi.MarshalReq(r, &payload); !ok {
		return hresp
	}

	tokenType := payload.TokenType
	if tokenType == "" {
		tokenType = "Bearer"
	}

	// Verify the token by asking Discord who it belongs to. A forged or
	// expired token fails here, so the identity we mint a session for is
	// always one the caller genuinely holds.
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

	// find-or-create the Metro user this Discord account links to, then
	// sync its profile fields — same steps the panel callback runs.
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

	// Best-effort role sync: this is what sets/clears is_staff, so a staff
	// member logging in via the website gets the same standing they'd get
	// via the panel. A non-member (or the bot being offline) just yields no
	// roles, which is fine for an ordinary reviewer.
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

	sessionToken := crypto.RandString(64)
	expiresAt := time.Now().Add(sessionTTL)

	if _, err := state.Pool.Exec(d.Context, `
		UPDATE discord_accounts SET session_token = $1, session_expires_at = $2
		WHERE discord_id = $3`,
		sessionToken, expiresAt, discordID,
	); err != nil {
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
