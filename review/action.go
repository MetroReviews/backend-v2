package review

import (
	"context"
	"errors"

	"github.com/MetroReviews/backend-v2/cache"
	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/MetroReviews/backend-v2/webhooks"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type actionDef struct {
	AllowedStates []types.State
	Error         string
	NewState      types.State
}

var actions = map[types.Action]actionDef{
	types.ActionClaim: {
		AllowedStates: []types.State{types.StatePending},
		Error:         "This cannot be claimed as it is not pending review — maybe someone already claimed it?",
		NewState:      types.StateUnderReview,
	},
	types.ActionUnclaim: {
		AllowedStates: []types.State{types.StateUnderReview},
		Error:         "This cannot be unclaimed as it is not under review.",
		NewState:      types.StatePending,
	},
	types.ActionApprove: {
		AllowedStates: []types.State{types.StateUnderReview},
		Error:         "This cannot be approved as it is not under review.",
		NewState:      types.StateApproved,
	},
	types.ActionDeny: {
		AllowedStates: []types.State{types.StateUnderReview},
		Error:         "This cannot be denied as it is not under review.",
		NewState:      types.StateDenied,
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

type Result struct {
	OK      bool
	Message string
}

func ApplyBusinessAction(ctx context.Context, businessID uuid.UUID, action types.Action, reason string, reviewer uuid.UUID) Result {
	def, ok := actions[action]
	if !ok {
		return Result{Message: "Unknown action"}
	}
	if len(reason) < 5 {
		return Result{Message: "Reason must be at least 5 characters"}
	}

	var businessState types.State
	err := state.Pool.QueryRow(ctx, "SELECT status FROM businesses WHERE id = $1", businessID).Scan(&businessState)
	if errors.Is(err, pgx.ErrNoRows) {
		return Result{Message: "Business not found"}
	}
	if err != nil {
		return Result{Message: "Failed to load business: " + err.Error()}
	}

	if !stateAllowed(def.AllowedStates, businessState) {
		return Result{Message: def.Error}
	}

	tx, err := state.Pool.Begin(ctx)
	if err != nil {
		return Result{Message: "Failed to start transaction: " + err.Error()}
	}
	defer tx.Rollback(ctx)

	if action == types.ActionClaim {
		if _, err := tx.Exec(ctx,
			"UPDATE businesses SET status = $1, reviewer = $2, updated_at = NOW() WHERE id = $3", def.NewState, reviewer, businessID,
		); err != nil {
			return Result{Message: "Failed to update business: " + err.Error()}
		}
	} else if _, err := tx.Exec(ctx,
		"UPDATE businesses SET status = $1, updated_at = NOW() WHERE id = $2", def.NewState, businessID,
	); err != nil {
		return Result{Message: "Failed to update business: " + err.Error()}
	}

	res := finishAction(ctx, tx, "business", businessID.String(), action, reason, reviewer)
	if res.OK {
		InvalidateBusinessCache(ctx, businessID)
	}
	return res
}

func InvalidateBusinessCache(ctx context.Context, businessID uuid.UUID) {
	var slug string
	if err := state.Pool.QueryRow(ctx, "SELECT slug FROM businesses WHERE id = $1", businessID).Scan(&slug); err != nil {
		return
	}
	_ = cache.Del(ctx, "biz:detail:"+slug)
}

func ApplyProjectAction(ctx context.Context, projectID uuid.UUID, action types.Action, reason string, reviewer uuid.UUID) Result {
	def, ok := actions[action]
	if !ok {
		return Result{Message: "Unknown action"}
	}
	if len(reason) < 5 {
		return Result{Message: "Reason must be at least 5 characters"}
	}

	var projectState types.State
	err := state.Pool.QueryRow(ctx, "SELECT status FROM projects WHERE id = $1", projectID).Scan(&projectState)
	if errors.Is(err, pgx.ErrNoRows) {
		return Result{Message: "Project not found"}
	}
	if err != nil {
		return Result{Message: "Failed to load project: " + err.Error()}
	}

	if !stateAllowed(def.AllowedStates, projectState) {
		return Result{Message: def.Error}
	}

	tx, err := state.Pool.Begin(ctx)
	if err != nil {
		return Result{Message: "Failed to start transaction: " + err.Error()}
	}
	defer tx.Rollback(ctx)

	if action == types.ActionClaim {
		if _, err := tx.Exec(ctx,
			"UPDATE projects SET status = $1, reviewer = $2, updated_at = NOW() WHERE id = $3", def.NewState, reviewer, projectID,
		); err != nil {
			return Result{Message: "Failed to update project: " + err.Error()}
		}
	} else if _, err := tx.Exec(ctx,
		"UPDATE projects SET status = $1, updated_at = NOW() WHERE id = $2", def.NewState, projectID,
	); err != nil {
		return Result{Message: "Failed to update project: " + err.Error()}
	}

	return finishAction(ctx, tx, "project", projectID.String(), action, reason, reviewer)
}

func recordAction(ctx context.Context, tx pgx.Tx, targetType, targetID string, action types.Action, reason string, reviewer uuid.UUID) error {
	if _, err := tx.Exec(ctx,
		"INSERT INTO moderation_actions (target_type, target_id, action, reason, reviewer) VALUES ($1, $2, $3, $4, $5)",
		targetType, targetID, action, reason, reviewer,
	); err != nil {
		return errors.New("failed to record action: " + err.Error())
	}
	return nil
}

func finishAction(ctx context.Context, tx pgx.Tx, targetType, targetID string, action types.Action, reason string, reviewer uuid.UUID) Result {
	if err := recordAction(ctx, tx, targetType, targetID, action, reason, reviewer); err != nil {
		return Result{Message: err.Error()}
	}

	if err := tx.Commit(ctx); err != nil {
		return Result{Message: "Failed to commit: " + err.Error()}
	}

	webhooks.DispatchQueueAction(targetType, targetID, action, reason, reviewer)

	return Result{OK: true, Message: "Done"}
}
