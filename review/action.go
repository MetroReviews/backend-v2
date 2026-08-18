// Package review implements the shared moderation queue: the
// claim -> approve/deny state machine a submitted bot, business or project
// moves through before it's publicly visible, surfaced together in the
// Discord /queue command. This replaces silverpelt, which did the same
// state machine for bots only and then fanned the result out over
// webhooks to every other enrolled list. There's no fixed list of other
// lists to fan out to anymore, but the shape came back: every action's
// shared finishAction tail fires the matching event through the webhooks
// package, so anyone can register their own webhook against a subject and
// hear about it — see that package for delivery/signing.
package review

import (
	"context"
	"errors"
	"strconv"

	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/MetroReviews/backend-v2/webhooks"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// actionDef describes what a review action needs: which states it may be
// applied from, the error shown when it can't, and the state it transitions
// to. Shared by bots and businesses — the queue mechanics are identical,
// only the underlying table differs.
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

// Result is the outcome of applying one review action to one bot or business.
type Result struct {
	OK      bool
	Message string // human-readable outcome; set on both success and failure
}

// ApplyBotAction validates and applies a claim/unclaim/approve/deny to a
// bot, recording it in moderation_actions. reviewer is the acting staff
// member's Metro user id (types.User.ID) — never a raw Discord ID; callers
// that only have a Discord ID (the Discord bot's /queue commands) resolve
// it via identity.EnsureDiscordUser first. Callers are expected to have
// already checked that reviewer belongs to a staff member.
func ApplyBotAction(ctx context.Context, botID int64, action types.Action, reason string, reviewer uuid.UUID) Result {
	def, ok := actions[action]
	if !ok {
		return Result{Message: "Unknown action"}
	}
	if len(reason) < 5 {
		return Result{Message: "Reason must be at least 5 characters"}
	}

	var botState types.State
	err := state.Pool.QueryRow(ctx, "SELECT state FROM bots WHERE bot_id = $1", botID).Scan(&botState)
	if errors.Is(err, pgx.ErrNoRows) {
		return Result{Message: "Bot not found"}
	}
	if err != nil {
		return Result{Message: "Failed to load bot: " + err.Error()}
	}

	if !stateAllowed(def.AllowedStates, botState) {
		return Result{Message: def.Error}
	}

	tx, err := state.Pool.Begin(ctx)
	if err != nil {
		return Result{Message: "Failed to start transaction: " + err.Error()}
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op once committed

	if action == types.ActionClaim {
		if _, err := tx.Exec(ctx,
			"UPDATE bots SET state = $1, reviewer = $2 WHERE bot_id = $3", def.NewState, reviewer, botID,
		); err != nil {
			return Result{Message: "Failed to update bot: " + err.Error()}
		}
	} else if _, err := tx.Exec(ctx, "UPDATE bots SET state = $1 WHERE bot_id = $2", def.NewState, botID); err != nil {
		return Result{Message: "Failed to update bot: " + err.Error()}
	}

	return finishAction(ctx, tx, "bot", strconv.FormatInt(botID, 10), action, reason, reviewer)
}

// ApplyBusinessAction is ApplyBotAction's counterpart for businesses.
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
	defer tx.Rollback(ctx) //nolint:errcheck // no-op once committed

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

	return finishAction(ctx, tx, "business", businessID.String(), action, reason, reviewer)
}

// ApplyProjectAction is ApplyBotAction's counterpart for projects — a
// business's posted portfolio items, which go through the same queue.
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
	defer tx.Rollback(ctx) //nolint:errcheck // no-op once committed

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

// finishAction is every ApplyXAction's shared tail: record the
// moderation_actions row, commit, and — only once that's actually landed —
// fire the matching webhooks.EventQueue* event. Centralizing this here
// means a future ApplyXAction (a new reviewable subject type) gets
// webhook delivery for free just by ending with this same call, the same
// way it already gets moderation_actions logging for free.
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
