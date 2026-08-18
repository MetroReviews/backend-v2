package roles

import (
	"context"

	"github.com/MetroReviews/backend-v2/perms"
	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// AssignUser grants userID roleID, a no-op if they already hold it.
func AssignUser(ctx context.Context, userID, roleID uuid.UUID) error {
	_, err := state.Pool.Exec(ctx, `
		INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2)
		ON CONFLICT DO NOTHING`, userID, roleID)
	return err
}

// UnassignUser revokes roleID from userID, a no-op if they didn't hold it.
func UnassignUser(ctx context.Context, userID, roleID uuid.UUID) error {
	_, err := state.Pool.Exec(ctx, "DELETE FROM user_roles WHERE user_id = $1 AND role_id = $2", userID, roleID)
	return err
}

// MemberCount returns how many users currently hold roleID — for display
// only (e.g. warning before a delete), so it's a plain count rather than
// returning the full member list.
func MemberCount(ctx context.Context, roleID uuid.UUID) (int, error) {
	var count int
	err := state.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM user_roles WHERE role_id = $1", roleID).Scan(&count)
	return count, err
}

// UserRoles returns every role userID holds.
func UserRoles(ctx context.Context, userID uuid.UUID) ([]types.Role, error) {
	rows, err := state.Pool.Query(ctx, `
		SELECT r.id, r.name, r.discord_role_id, r.permissions, r.created_at, r.updated_at
		FROM roles r JOIN user_roles ur ON ur.role_id = r.id
		WHERE ur.user_id = $1
		ORDER BY r.name`, userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[types.Role])
}

// UserPermissions returns userID's effective permissions: the union of
// every role they hold.
func UserPermissions(ctx context.Context, userID uuid.UUID) ([]string, error) {
	roles, err := UserRoles(ctx, userID)
	if err != nil {
		return nil, err
	}
	sets := make([][]string, len(roles))
	for i, r := range roles {
		sets[i] = r.Permissions
	}
	return perms.Union(sets...), nil
}

// HasPermission reports whether userID's effective permissions include want.
func HasPermission(ctx context.Context, userID uuid.UUID, want string) (bool, error) {
	granted, err := UserPermissions(ctx, userID)
	if err != nil {
		return false, err
	}
	return perms.Has(granted, want), nil
}
