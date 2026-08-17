package panel

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/MetroReviews/backend-v2/state"
	"github.com/infinitybotlist/eureka/uapi"
)

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
