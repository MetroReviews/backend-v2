package panel

import (
	"encoding/base64"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/MetroReviews/backend-v2/helpers"
	"github.com/MetroReviews/backend-v2/identity"
	"github.com/MetroReviews/backend-v2/roles"
	"github.com/MetroReviews/backend-v2/state"
	"github.com/infinitybotlist/eureka/crypto"
	"github.com/infinitybotlist/eureka/jsonimpl"
	"github.com/infinitybotlist/eureka/uapi"
)

func completeOAuth2(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	code := r.URL.Query().Get("code")
	redirectState := r.URL.Query().Get("state")

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("client_id", appID())
	form.Set("redirect_uri", state.Config.Auth.OAuthRedirect)
	form.Set("client_secret", state.Config.Discord.ClientSecret)

	tokenResp, err := http.PostForm("https://discord.com/api/v10/oauth2/token", form)
	if err != nil {
		return helpers.ErrorResponse(http.StatusBadGateway, err.Error())
	}
	defer tokenResp.Body.Close()

	var token struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		Scope       string `json:"scope"`
	}
	if err := jsonimpl.UnmarshalReader(tokenResp.Body, &token); err != nil || tokenResp.StatusCode != http.StatusOK {
		return helpers.ErrorResponse(http.StatusBadRequest, "OAuth2 token exchange failed")
	}

	scopes := strings.Fields(token.Scope)
	if !helpers.Contains(scopes, "identify") || !helpers.Contains(scopes, "guilds") {
		return helpers.ErrorResponse(http.StatusBadRequest, "Invalid scopes, got "+token.Scope)
	}
	if token.TokenType != "Bearer" {
		return helpers.ErrorResponse(http.StatusBadRequest, "Invalid token type, got "+token.TokenType)
	}

	req, _ := http.NewRequestWithContext(d.Context, http.MethodGet, "https://discord.com/api/v10/users/@me", nil)
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	userResp, err := http.DefaultClient.Do(req)
	if err != nil {
		return helpers.ErrorResponse(http.StatusBadGateway, err.Error())
	}
	defer userResp.Body.Close()

	var user struct {
		ID            string `json:"id"`
		Username      string `json:"username"`
		Discriminator string `json:"discriminator"`
		Avatar        string `json:"avatar"`
	}
	if err := jsonimpl.UnmarshalReader(userResp.Body, &user); err != nil || userResp.StatusCode != http.StatusOK {
		return helpers.ErrorResponse(http.StatusBadRequest, "Failed to fetch user")
	}

	nonce := crypto.RandString(43) + "@" + strconv.FormatFloat(float64(time.Now().UnixNano())/1e9, 'f', -1, 64)

	discordID, _ := strconv.ParseInt(user.ID, 10, 64)

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
		if member, err := state.Discord.GuildMember(gid, user.ID); err == nil && member != nil {
			memberRoles = member.Roles
		}
	}
	if err := roles.SyncMember(d.Context, metroUserID, discordID, memberRoles); err != nil {
		return helpers.InternalError(err)
	}

	if _, err := state.Pool.Exec(d.Context,
		"UPDATE discord_accounts SET nonce = $1 WHERE discord_id = $2", nonce, discordID,
	); err != nil {
		return helpers.InternalError(err)
	}

	sessionToken, _, err := identity.NewSession(d.Context, metroUserID)
	if err != nil {
		return helpers.InternalError(err)
	}

	ticketData := map[string]string{
		"nonce":         nonce,
		"user_id":       user.ID,
		"username":      user.Username,
		"disc":          user.Discriminator,
		"avatar":        user.Avatar,
		"session_token": sessionToken,
	}
	ticketBytes, _ := jsonimpl.Marshal(ticketData)
	ticket := base64.URLEncoding.EncodeToString(ticketBytes)

	return uapi.HttpResponse{Redirect: redirectState + "/login?ticket=" + ticket}
}
