package webhooks

import (
	"context"
	"errors"
	"strconv"

	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// OwnsTarget reports whether user owns targetType/targetID — the same
// rule that gates who may register/manage a webhook for it. Mirrors the
// per-subject ownership checks already scattered across routes/projects
// and routes/reviews (a business's owner_id; a project's owning
// business's owner_id; a bot's owner/extra_owners, which are raw Discord
// IDs), generalized to string target IDs since a webhook only knows its
// target that way. Callers should let staff (user.IsStaff) bypass this
// separately — it only encodes the ownership rule itself.
//
// An unrecognized target type reports false rather than erroring: there's
// no ownership rule for it (yet), so only staff can manage its webhooks
// until one exists.
func OwnsTarget(ctx context.Context, user *types.User, targetType, targetID string) (bool, error) {
	switch targetType {
	case "business":
		return ownsBusiness(ctx, user, targetID)
	case "project":
		return ownsProject(ctx, user, targetID)
	case "bot":
		return ownsBot(ctx, user, targetID)
	default:
		return false, nil
	}
}

func ownsBusiness(ctx context.Context, user *types.User, targetID string) (bool, error) {
	id, err := uuid.Parse(targetID)
	if err != nil {
		return false, nil
	}
	var ownerID *uuid.UUID
	err = state.Pool.QueryRow(ctx, "SELECT owner_id FROM businesses WHERE id = $1", id).Scan(&ownerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return ownerID != nil && *ownerID == user.ID, nil
}

func ownsProject(ctx context.Context, user *types.User, targetID string) (bool, error) {
	id, err := uuid.Parse(targetID)
	if err != nil {
		return false, nil
	}
	var ownerID *uuid.UUID
	err = state.Pool.QueryRow(ctx, `
		SELECT b.owner_id FROM projects p JOIN businesses b ON b.id = p.business_id WHERE p.id = $1`, id,
	).Scan(&ownerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return ownerID != nil && *ownerID == user.ID, nil
}

func ownsBot(ctx context.Context, user *types.User, targetID string) (bool, error) {
	if user.DiscordID == nil {
		return false, nil
	}
	botID, err := strconv.ParseInt(targetID, 10, 64)
	if err != nil {
		return false, nil
	}
	var owner int64
	var extraOwners []int64
	err = state.Pool.QueryRow(ctx, "SELECT owner, extra_owners FROM bots WHERE bot_id = $1", botID).Scan(&owner, &extraOwners)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if owner == *user.DiscordID {
		return true, nil
	}
	for _, id := range extraOwners {
		if id == *user.DiscordID {
			return true, nil
		}
	}
	return false, nil
}
