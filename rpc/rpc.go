// Package rpc is the single place review-queue actions (review.go) and
// role-management actions (roles.go) actually execute through, regardless
// of whether the caller is the HTTP API or the Discord bot. Both surfaces
// used to call straight into the review/roles domain packages and do their
// own authorization and result-handling around that call, which let the
// two drift — this package exists so there's exactly one authorization
// check and exactly one audit trail per action, no matter which surface
// triggered it.
//
// The review/roles domain packages themselves aren't changed by this —
// this is purely the authorization+audit wrapper in front of them.
package rpc

import (
	"context"
	"errors"
	"fmt"

	"github.com/MetroReviews/backend-v2/roles"
	"github.com/MetroReviews/backend-v2/state"
	"github.com/google/uuid"
)

// Actor is who's calling an rpc function — a resolved Metro user (never a
// raw Discord ID), tagged with where the call came from so the audit trail
// can say so.
type Actor struct {
	UserID    uuid.UUID // see identity.EnsureDiscordUser (Discord) / api.AuthUser (HTTP)
	DiscordID *int64    // for audit-log mentions; nil is handled gracefully
	Source    string    // "api" or "discord"
}

// ForbiddenError is returned when actor doesn't hold the permission an
// rpc function requires.
type ForbiddenError struct{ Permission string }

func (e *ForbiddenError) Error() string {
	return "missing required permission: " + e.Permission
}

// ErrDiscordUnavailable is returned by rpc functions that need a live
// Discord session (SyncGuildRoles) when the bot isn't connected.
var ErrDiscordUnavailable = errors.New("the Discord bot isn't connected")

// authorize checks actor holds permission, honoring the same config-owner
// bypass every existing permission check in this codebase does.
func authorize(ctx context.Context, actor Actor, permission string) error {
	if actor.DiscordID != nil && state.Config.IsOwner(*actor.DiscordID) {
		return nil
	}
	has, err := roles.HasPermission(ctx, actor.UserID, permission)
	if err != nil {
		return err
	}
	if !has {
		return &ForbiddenError{Permission: permission}
	}
	return nil
}

// mentionActor formats actor for an audit-log embed field.
func mentionActor(actor Actor) string {
	if actor.DiscordID != nil {
		return fmt.Sprintf("<@%d>", *actor.DiscordID)
	}
	return "an unknown user"
}
