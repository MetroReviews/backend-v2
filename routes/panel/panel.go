package panel

import (
	"encoding/base64"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/go-chi/chi/v5"
	"github.com/infinitybotlist/eureka/crypto"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/jsonimpl"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
)

const tagName = "Panel (Internal)"

type Router struct{}

func (Router) Tag() (string, string) {
	return tagName, "Internal endpoints used by the Metro Reviews panel."
}

type oauthURLResponse struct {
	URL string `json:"url" description:"The Discord OAuth2 authorize URL"`
}

type panelMember struct {
	ID   string `json:"id" description:"The member's Discord ID"`
	Name string `json:"name" description:"The member's username"`
}

type panelAccessResponse struct {
	Access bool         `json:"access" description:"Whether panel access is granted"`
	Hint   string       `json:"hint,omitempty" description:"A hint about why access was denied"`
	Member *panelMember `json:"member,omitempty" description:"The authenticated member"`
}

func (Router) Routes(r *chi.Mux) {
	uapi.Route{
		Method:  uapi.GET,
		Pattern: "/_panel/strikestone",
		OpId:    "get_oauth2",
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Get OAuth2 URL",
				Description: "Returns the Discord OAuth2 authorize URL for the review panel.",
				Resp:        oauthURLResponse{},
			}
		},
		Handler: getOAuth2,
	}.Route(r)

	uapi.Route{
		Method:  uapi.GET,
		Pattern: "/_panel/mapleshade",
		OpId:    "get_panel_access",
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Get Panel Access",
				Description: "Verifies a panel access ticket and returns whether the holder may access the panel.",
				Resp:        panelAccessResponse{},
				Params: []docs.Parameter{
					{Name: "ticket", In: "query", Description: "The base64url-encoded ticket", Required: true, Schema: docs.IdSchema},
				},
			}
		},
		Handler: getPanelAccess,
	}.Route(r)

	uapi.Route{
		Method:  uapi.GET,
		Pattern: "/_panel/frostpaw",
		OpId:    "complete_oauth2",
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Complete OAuth2",
				Description: "Completes the Discord OAuth2 handshake and redirects back to the panel with a ticket.",
				Resp:        types.ApiError{},
				Params: []docs.Parameter{
					{Name: "code", In: "query", Description: "The OAuth2 authorization code", Required: true, Schema: docs.IdSchema},
					{Name: "state", In: "query", Description: "The origin to redirect back to", Required: true, Schema: docs.IdSchema},
				},
			}
		},
		Handler: completeOAuth2,
	}.Route(r)
}

func checkNonceTime(nonce string) bool {
	split := strings.Split(nonce, "@")
	if len(split) != 2 {
		return false
	}
	t, err := strconv.ParseFloat(split[1], 64)
	if err != nil {
		return false
	}
	return float64(time.Now().Unix())-t <= 60*30
}

func appID() string {
	if state.Discord != nil && state.Discord.State != nil && state.Discord.State.User != nil {
		return state.Discord.State.User.ID
	}
	return ""
}

func getOAuth2(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	origin := r.Header.Get("Origin")
	if origin == "" {
		origin = state.Config.DefaultLoginState
	}

	authURL := "https://discord.com/api/oauth2/authorize?client_id=" + appID() +
		"&permissions=0&scope=identify%20guilds&response_type=code&redirect_uri=" +
		url.QueryEscape(state.Config.OAuthRedirect) + "&state=" + origin

	return uapi.HttpResponse{Json: oauthURLResponse{URL: authURL}}
}

func getPanelAccess(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	raw := r.URL.Query().Get("ticket")

	decoded, err := base64.URLEncoding.DecodeString(raw)
	if err != nil {
		return uapi.HttpResponse{Json: panelAccessResponse{Access: false}}
	}

	var ticket struct {
		Nonce  string `json:"nonce"`
		UserID string `json:"user_id"`
	}
	if err := jsonimpl.Unmarshal(decoded, &ticket); err != nil {
		return uapi.HttpResponse{Json: panelAccessResponse{Access: false}}
	}

	if ticket.Nonce == "" || ticket.UserID == "" {
		return uapi.HttpResponse{Json: panelAccessResponse{Access: false}}
	}

	if !checkNonceTime(ticket.Nonce) {
		return uapi.HttpResponse{Json: panelAccessResponse{Access: false, Hint: "Nonce expiry"}}
	}

	userID, err := strconv.ParseInt(ticket.UserID, 10, 64)
	if err != nil {
		return uapi.HttpResponse{Json: panelAccessResponse{Access: false}}
	}

	var dbNonce string
	err = state.Pool.QueryRow(d.Context,
		"SELECT nonce FROM users WHERE user_id = $1 AND nonce = $2", userID, ticket.Nonce,
	).Scan(&dbNonce)
	if errors.Is(err, pgx.ErrNoRows) || err != nil {
		return uapi.HttpResponse{Json: panelAccessResponse{Access: false}}
	}

	gid := strconv.FormatUint(state.Config.GuildID(), 10)
	member, err := state.Discord.GuildMember(gid, ticket.UserID)
	if err != nil || member == nil {
		return uapi.HttpResponse{Json: panelAccessResponse{Access: false}}
	}

	isReviewer := state.Config.IsOwner(userID)
	for _, roleID := range member.Roles {
		if roleID == state.Config.Reviewer {
			isReviewer = true
		}
	}

	if !isReviewer {
		return uapi.HttpResponse{Json: panelAccessResponse{Access: false}}
	}

	return uapi.HttpResponse{Json: panelAccessResponse{
		Access: true,
		Member: &panelMember{ID: member.User.ID, Name: member.User.Username},
	}}
}

func completeOAuth2(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	code := r.URL.Query().Get("code")
	redirectState := r.URL.Query().Get("state")

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("client_id", appID())
	form.Set("redirect_uri", state.Config.OAuthRedirect)
	form.Set("client_secret", state.Config.ClientSecret)

	tokenResp, err := http.PostForm("https://discord.com/api/v10/oauth2/token", form)
	if err != nil {
		return uapi.HttpResponse{Status: http.StatusBadGateway, Json: types.ApiError{Message: err.Error(), Error: true}}
	}
	defer tokenResp.Body.Close()

	var token struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		Scope       string `json:"scope"`
	}
	if err := jsonimpl.UnmarshalReader(tokenResp.Body, &token); err != nil || tokenResp.StatusCode != http.StatusOK {
		return uapi.HttpResponse{Status: http.StatusBadRequest, Json: types.ApiError{Message: "OAuth2 token exchange failed", Error: true}}
	}

	scopes := strings.Fields(token.Scope)
	if !contains(scopes, "identify") || !contains(scopes, "guilds") {
		return uapi.HttpResponse{Status: http.StatusBadRequest, Json: types.ApiError{Message: "Invalid scopes, got " + token.Scope, Error: true}}
	}
	if token.TokenType != "Bearer" {
		return uapi.HttpResponse{Status: http.StatusBadRequest, Json: types.ApiError{Message: "Invalid token type, got " + token.TokenType, Error: true}}
	}

	req, _ := http.NewRequestWithContext(d.Context, http.MethodGet, "https://discord.com/api/v10/users/@me", nil)
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	userResp, err := http.DefaultClient.Do(req)
	if err != nil {
		return uapi.HttpResponse{Status: http.StatusBadGateway, Json: types.ApiError{Message: err.Error(), Error: true}}
	}
	defer userResp.Body.Close()

	var user struct {
		ID            string `json:"id"`
		Username      string `json:"username"`
		Discriminator string `json:"discriminator"`
		Avatar        string `json:"avatar"`
	}
	if err := jsonimpl.UnmarshalReader(userResp.Body, &user); err != nil || userResp.StatusCode != http.StatusOK {
		return uapi.HttpResponse{Status: http.StatusBadRequest, Json: types.ApiError{Message: "Failed to fetch user", Error: true}}
	}

	nonce := crypto.RandString(43) + "@" + strconv.FormatFloat(float64(time.Now().UnixNano())/1e9, 'f', -1, 64)

	userID, _ := strconv.ParseInt(user.ID, 10, 64)
	if _, err := state.Pool.Exec(d.Context,
		"INSERT INTO users (user_id, nonce) VALUES ($1, $2) ON CONFLICT (user_id) DO UPDATE SET nonce = EXCLUDED.nonce",
		userID, nonce,
	); err != nil {
		return uapi.HttpResponse{Status: http.StatusInternalServerError, Json: types.ApiError{Message: err.Error(), Error: true}}
	}

	ticketData := map[string]string{
		"nonce":    nonce,
		"user_id":  user.ID,
		"username": user.Username,
		"disc":     user.Discriminator,
		"avatar":   user.Avatar,
	}
	ticketBytes, _ := jsonimpl.Marshal(ticketData)
	ticket := base64.URLEncoding.EncodeToString(ticketBytes)

	return uapi.HttpResponse{Redirect: redirectState + "/login?ticket=" + ticket}
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
