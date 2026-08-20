package webhooks

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/google/uuid"
	"github.com/infinitybotlist/eureka/jsonimpl"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

const maxConsecutiveFailures = 10

var httpClient = &http.Client{Timeout: 10 * time.Second}

func Dispatch(targetType, targetID, event string, data any) {
	go func() {
		hooks, err := matchingWebhooks(state.Context, targetType, targetID, event)
		if err != nil {
			state.Logger.Error("[webhooks] failed to load webhooks for dispatch",
				zap.Error(err), zap.String("event", event), zap.String("target_type", targetType), zap.String("target_id", targetID))
			return
		}
		for _, hook := range hooks {
			go func(hook types.Webhook) {
				if err := deliver(hook, event, targetType, targetID, data); err != nil {
					state.Logger.Warn("[webhooks] delivery failed",
						zap.Error(err), zap.String("webhook_id", hook.ID.String()), zap.String("event", event))
					recordFailure(hook.ID)
					return
				}
				recordSuccess(hook.ID)
			}(hook)
		}
	}()
}

func DispatchQueueAction(targetType, targetID string, action types.Action, reason string, reviewer uuid.UUID) {
	event, ok := queueActionEvents[action]
	if !ok {
		return
	}
	Dispatch(targetType, targetID, event, map[string]any{
		"target_type": targetType,
		"target_id":   targetID,
		"action":      queueActionNames[action],
		"reason":      reason,
		"reviewer_id": reviewer,
	})
}

func DeliverTest(hook types.Webhook) error {
	return deliver(hook, EventTest, hook.TargetType, hook.TargetID, map[string]any{
		"message": "This is a test delivery from Metro Reviews.",
	})
}

func matchingWebhooks(ctx context.Context, targetType, targetID, event string) ([]types.Webhook, error) {
	rows, err := state.Pool.Query(ctx, `
		SELECT `+webhookColumns+` FROM webhooks
		WHERE target_type = $1 AND target_id = $2 AND enabled = TRUE
		  AND (events = '{}' OR events @> ARRAY[$3]::TEXT[])`,
		targetType, targetID, event,
	)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[types.Webhook])
}

func deliver(hook types.Webhook, event, targetType, targetID string, data any) error {
	deliveryID := uuid.NewString()

	body, err := jsonimpl.Marshal(map[string]any{
		"id":          deliveryID,
		"event":       event,
		"target_type": targetType,
		"target_id":   targetID,
		"created_at":  time.Now().UTC().Format(time.RFC3339),
		"data":        data,
	})
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	mac := hmac.New(sha256.New, []byte(hook.Secret))
	mac.Write(body)
	signature := hex.EncodeToString(mac.Sum(nil))

	req, err := http.NewRequest(http.MethodPost, hook.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Metro-Event", event)
	req.Header.Set("X-Metro-Delivery", deliveryID)
	req.Header.Set("X-Metro-Signature", "sha256="+signature)

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("receiver returned %s", resp.Status)
	}
	return nil
}

func recordSuccess(id uuid.UUID) {
	if _, err := state.Pool.Exec(state.Context, `
		UPDATE webhooks SET failure_count = 0, last_triggered_at = NOW(), updated_at = NOW() WHERE id = $1`, id,
	); err != nil {
		state.Logger.Error("[webhooks] failed to record delivery success", zap.Error(err), zap.String("webhook_id", id.String()))
	}
}

func recordFailure(id uuid.UUID) {
	var failureCount int
	err := state.Pool.QueryRow(state.Context, `
		UPDATE webhooks SET failure_count = failure_count + 1, updated_at = NOW()
		WHERE id = $1 RETURNING failure_count`, id,
	).Scan(&failureCount)
	if err != nil {
		state.Logger.Error("[webhooks] failed to record delivery failure", zap.Error(err), zap.String("webhook_id", id.String()))
		return
	}

	if failureCount < maxConsecutiveFailures {
		return
	}
	if _, err := state.Pool.Exec(state.Context, "UPDATE webhooks SET enabled = FALSE WHERE id = $1", id); err != nil {
		state.Logger.Error("[webhooks] failed to auto-disable webhook", zap.Error(err), zap.String("webhook_id", id.String()))
		return
	}
	state.Logger.Warn("[webhooks] auto-disabled webhook after repeated delivery failures", zap.String("webhook_id", id.String()))
}
