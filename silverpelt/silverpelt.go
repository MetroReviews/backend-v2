// Package silverpelt propagates review actions (claim/unclaim/approve/deny)
// from Metro Reviews out to every enrolled bot list's webhook.
package silverpelt

import (
	"context"
	"fmt"

	"github.com/MetroReviews/backend-v2/helpers"
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

type Response struct {
	Message string                  `json:"message,omitempty"`
	Lists   map[string]HTTPResponse `json:"lists,omitempty"`
}

func (r *Response) ToHTML() string {
	b, _ := jsonimpl.Marshal(r)
	return string(b)
}

type listRow struct {
	ID        uuid.UUID
	Name      string
	Domain    *string
	State     types.ListState
	URL       string
	SecretKey string
}

// Handle applies a review action to a bot and dispatches it to every
// enrolled, eligible list's webhook, returning a per-list status summary.
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
		state.Logger.Error("[silverpelt] failed to load lists", zap.Error(err))
		return &Response{Message: "Failed to load lists."}
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

		if len(data.Lists) > 0 && !helpers.Contains(data.Lists, l.ID.String()) {
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
