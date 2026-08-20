package rpc

import (
	"context"
	"errors"
	"fmt"

	"github.com/MetroReviews/backend-v2/roles"
	"github.com/MetroReviews/backend-v2/state"
	"github.com/google/uuid"
)

type Actor struct {
	UserID    uuid.UUID
	DiscordID *int64
	Source    string
}

type ForbiddenError struct{ Permission string }

func (e *ForbiddenError) Error() string {
	return "missing required permission: " + e.Permission
}

var ErrDiscordUnavailable = errors.New("the Discord bot isn't connected")

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

func mentionActor(actor Actor) string {
	if actor.DiscordID != nil {
		return fmt.Sprintf("<@%d>", *actor.DiscordID)
	}
	return "an unknown user"
}
