package silverpelt

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/infinitybotlist/eureka/jsonimpl"
	"go.uber.org/zap"
)

// actionDef describes what a review action needs: which states it may be
// applied from, the error shown when it can't, the state it transitions
// to, and the bot_list column holding the webhook URL to call.
type actionDef struct {
	AllowedStates []types.State
	Error         string
	NewState      types.State
	ListColumn    string
}

var actions = map[types.Action]actionDef{
	types.ActionClaim: {
		AllowedStates: []types.State{types.StatePending},
		Error:         "This bot cannot be claimed as it is not pending review? Maybe someone is testing it right now?",
		NewState:      types.StateUnderReview,
		ListColumn:    "claim_bot_api",
	},
	types.ActionUnclaim: {
		AllowedStates: []types.State{types.StateUnderReview},
		Error:         "This bot cannot be unclaimed as it is not under review?",
		NewState:      types.StatePending,
		ListColumn:    "unclaim_bot_api",
	},
	types.ActionApprove: {
		AllowedStates: []types.State{types.StateUnderReview},
		Error:         "This bot cannot be approved as it is not under review?",
		NewState:      types.StateApproved,
		ListColumn:    "approve_bot_api",
	},
	types.ActionDeny: {
		AllowedStates: []types.State{types.StateUnderReview},
		Error:         "This bot cannot be denied as it is not under review?",
		NewState:      types.StateDenied,
		ListColumn:    "deny_bot_api",
	},
}

func stateAllowed(allowed []types.State, s types.State) bool {
	for _, a := range allowed {
		if a == s {
			return true
		}
	}
	return false
}

// HTTPResponse records the outcome of dispatching one action to one list's
// webhook, in the shape lists expect back from the review-action HTML/JSON
// response.
type HTTPResponse struct {
	Status   int            `json:"status"`
	Msg      string         `json:"msg,omitempty"`
	Data     string         `json:"data,omitempty"`
	Exc      string         `json:"exc,omitempty"`
	SentData map[string]any `json:"sent_data"`
}

var httpClient = &http.Client{Timeout: 30 * time.Second}

// makeRequest posts payload to a list's webhook URL and reports what
// happened. Errors are captured in the returned HTTPResponse rather than
// as a Go error, since a failed dispatch to one list is a normal, expected
// per-list result rather than a fatal error for the whole request.
func makeRequest(ctx context.Context, url, key string, payload map[string]any) HTTPResponse {
	if url == "" || !strings.HasPrefix(url, "https://") {
		return HTTPResponse{Status: 400, Msg: "No url provided", SentData: payload}
	}

	state.Logger.Info("[silverpelt] dispatching webhook", zap.String("url", url))

	body, err := jsonimpl.Marshal(payload)
	if err != nil {
		return HTTPResponse{Status: 400, Msg: "Failed to encode payload", Exc: err.Error(), SentData: payload}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(body)))
	if err != nil {
		return HTTPResponse{Status: 400, Msg: "Failed to build request", Exc: err.Error(), SentData: payload}
	}
	req.Header.Set("Authorization", key)
	req.Header.Set("User-Agent", "Frostpaw/0.2 (Silverpelt)")
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return HTTPResponse{Status: 400, Msg: "Request failed", Exc: err.Error(), SentData: payload}
	}
	defer resp.Body.Close()

	var sb strings.Builder
	buf := make([]byte, 4096)
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			sb.Write(buf[:n])
		}
		if rerr != nil {
			break
		}
	}

	return HTTPResponse{
		Status:   resp.StatusCode,
		Msg:      "Success",
		Data:     sb.String(),
		SentData: payload,
	}
}
