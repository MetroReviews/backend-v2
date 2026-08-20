package webhooks

import (
	"context"
	"fmt"
	"strings"

	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/google/uuid"
	"github.com/infinitybotlist/eureka/crypto"
	"github.com/jackc/pgx/v5"
)

const webhookColumns = `id, target_type, target_id, url, secret, events, created_by, enabled, failure_count, last_triggered_at, created_at, updated_at`

func scanWebhook(row pgx.Row) (*types.Webhook, error) {
	var w types.Webhook
	if err := row.Scan(
		&w.ID, &w.TargetType, &w.TargetID, &w.URL, &w.Secret, &w.Events,
		&w.CreatedBy, &w.Enabled, &w.FailureCount, &w.LastTriggeredAt, &w.CreatedAt, &w.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &w, nil
}

func List(ctx context.Context, targetType, targetID string) ([]types.Webhook, error) {
	rows, err := state.Pool.Query(ctx,
		"SELECT "+webhookColumns+" FROM webhooks WHERE target_type = $1 AND target_id = $2 ORDER BY created_at", targetType, targetID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[types.Webhook])
}

func Get(ctx context.Context, id uuid.UUID) (*types.Webhook, error) {
	return scanWebhook(state.Pool.QueryRow(ctx, "SELECT "+webhookColumns+" FROM webhooks WHERE id = $1", id))
}

func Create(ctx context.Context, targetType, targetID, url string, events []string, createdBy uuid.UUID) (*types.Webhook, error) {
	if events == nil {
		events = []string{}
	}
	secret := crypto.RandString(48)
	return scanWebhook(state.Pool.QueryRow(ctx, `
		INSERT INTO webhooks (target_type, target_id, url, secret, events, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING `+webhookColumns,
		targetType, targetID, url, secret, events, createdBy,
	))
}

func Update(ctx context.Context, id uuid.UUID, url *string, events []string, enabled *bool) (*types.Webhook, error) {
	var setClauses []string
	var args []any

	set := func(column string, value any) {
		args = append(args, value)
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", column, len(args)))
	}

	if url != nil {
		set("url", *url)
	}
	if events != nil {
		set("events", events)
	}
	if enabled != nil {
		set("enabled", *enabled)
	}

	if len(setClauses) == 0 {
		return Get(ctx, id)
	}
	setClauses = append(setClauses, "updated_at = NOW()")

	args = append(args, id)
	query := fmt.Sprintf("UPDATE webhooks SET %s WHERE id = $%d RETURNING %s",
		strings.Join(setClauses, ", "), len(args), webhookColumns)
	return scanWebhook(state.Pool.QueryRow(ctx, query, args...))
}

func Delete(ctx context.Context, id uuid.UUID) error {
	_, err := state.Pool.Exec(ctx, "DELETE FROM webhooks WHERE id = $1", id)
	return err
}

func RotateSecret(ctx context.Context, id uuid.UUID) (*types.Webhook, error) {
	return scanWebhook(state.Pool.QueryRow(ctx, `
		UPDATE webhooks SET secret = $1, updated_at = NOW() WHERE id = $2
		RETURNING `+webhookColumns,
		crypto.RandString(48), id,
	))
}
