package silverpelt

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/google/uuid"
	"github.com/infinitybotlist/eureka/jsonimpl"
	"go.uber.org/zap"
)

type Request struct {
	BotID    int64
	Reason   string
	Resend   bool
	Action   types.Action
	Reviewer int64
	Lists    []string
}

type actionDef struct {
	AllowedStates []types.State
	Error         string
	NewState      types.State
	ListColumn    string
}

type HTTPResponse struct {
	Status   int            `json:"status"`
	Msg      string         `json:"msg,omitempty"`
	Data     string         `json:"data,omitempty"`
	Exc      string         `json:"exc,omitempty"`
	SentData map[string]any `json:"sent_data"`
}

type Response struct {
	Message string                  `json:"message,omitempty"`
	Lists   map[string]HTTPResponse `json:"lists,omitempty"`
}

func (r *Response) ToMsg() string {
	var b strings.Builder
	if r.Message != "" {
		b.WriteString("**Request Failed**\n")
		b.WriteString(r.Message)
	}
	if r.Lists != nil {
		var parts []string
		for name, resp := range r.Lists {
			msg := resp.Msg
			if msg == "" {
				msg = "No error: "
			}
			data := resp.Data
			if data == "" {
				data = "No data"
			} else if len(data) > 50 {
				data = data[:50]
			}
			parts = append(parts, fmt.Sprintf("%s: %s %s", name, msg, data))
		}
		b.WriteString("\n" + strings.Join(parts, "\n"))
	}
	out := b.String()
	if len(out) > 1900 {
		out = out[:1900]
	}
	return out + "...<some lines omitted>"
}

func (r *Response) ToHTML() string {
	b, _ := jsonimpl.Marshal(r)
	return string(b)
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

var httpClient = &http.Client{Timeout: 30 * time.Second}

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

type listRow struct {
	ID        uuid.UUID
	Name      string
	Domain    *string
	State     types.ListState
	URL       string
	SecretKey string
}

func Handle(ctx context.Context, data Request) *Response {
	if len(data.Reason) < 5 {
		return &Response{Message: "Reason must be at least 5 characters"}
	}

	var botState types.State
	var listSource uuid.UUID
	var crossAdd bool
	err := state.Pool.QueryRow(ctx,
		"SELECT state, list_source, cross_add FROM bot_queue WHERE bot_id = $1",
		data.BotID,
	).Scan(&botState, &listSource, &crossAdd)
	if err != nil {
		return &Response{Message: "Bot not found"}
	}

	action, ok := actions[data.Action]
	if !ok {
		return &Response{Message: "Unknown action"}
	}

	if !data.Resend {
		if !stateAllowed(action.AllowedStates, botState) {
			return &Response{Message: action.Error}
		}
	}

	if data.Action == types.ActionClaim && !data.Resend {
		if _, err := state.Pool.Exec(ctx,
			"UPDATE bot_queue SET reviewer = $1 WHERE bot_id = $2", data.Reviewer, data.BotID,
		); err != nil {
			state.Logger.Error("[silverpelt] failed to set reviewer", zap.Error(err))
		}
	}

	if _, err := state.Pool.Exec(ctx,
		"UPDATE bot_queue SET state = $1 WHERE bot_id = $2", action.NewState, data.BotID,
	); err != nil {
		state.Logger.Error("[silverpelt] failed to update state", zap.Error(err))
	}

	query := fmt.Sprintf(
		"SELECT id, name, domain, state, %s, secret_key FROM bot_list", action.ListColumn,
	)
	rows, err := state.Pool.Query(ctx, query)
	if err != nil {
		return &Response{Message: "Failed to load lists: " + err.Error()}
	}
	defer rows.Close()

	var lists []listRow
	for rows.Next() {
		var l listRow
		if err := rows.Scan(&l.ID, &l.Name, &l.Domain, &l.State, &l.URL, &l.SecretKey); err != nil {
			state.Logger.Error("[silverpelt] failed to scan list", zap.Error(err))
			continue
		}
		lists = append(lists, l)
	}

	listResp := make(map[string]HTTPResponse)

	for _, l := range lists {
		name := l.ID.String()
		if l.Domain != nil && *l.Domain != "" {
			name = *l.Domain
		} else if l.Name != "" {
			name = l.Name
		}

		if !types.IsGoodListState(l.State) {
			continue
		}

		if len(data.Lists) > 0 && !contains(data.Lists, l.ID.String()) {
			continue
		}

		canAdd := true
		if l.ID != listSource && !crossAdd {
			canAdd = false
		}

		reason := data.Reason
		if reason == "" {
			reason = "No reason provided"
		}

		payload := map[string]any{
			"bot_id":   fmt.Sprintf("%d", data.BotID),
			"can_add":  canAdd,
			"reviewer": fmt.Sprintf("%d", data.Reviewer),
			"reason":   reason,
		}

		listResp[name] = makeRequest(ctx, l.URL, l.SecretKey, payload)
	}

	return &Response{Lists: listResp}
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
