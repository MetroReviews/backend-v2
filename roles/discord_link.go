package roles

import (
	"context"
	"strconv"

	"github.com/MetroReviews/backend-v2/perms"
	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/jackc/pgx/v5"
)

func DiscordRoleIDsWithPermission(ctx context.Context, want string) ([]int64, error) {
	rows, err := state.Pool.Query(ctx, `
		SELECT discord_role_id FROM roles
		WHERE discord_role_id IS NOT NULL
		  AND (permissions @> ARRAY[$1]::TEXT[] OR permissions @> ARRAY[$2]::TEXT[])`,
		want, perms.Wildcard,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func linkedRoles(ctx context.Context) (map[string]types.Role, error) {
	rows, err := state.Pool.Query(ctx, "SELECT "+roleColumns+" FROM roles WHERE discord_role_id IS NOT NULL")
	if err != nil {
		return nil, err
	}
	list, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.Role])
	if err != nil {
		return nil, err
	}

	out := make(map[string]types.Role, len(list))
	for _, role := range list {
		out[strconv.FormatInt(*role.DiscordRoleID, 10)] = role
	}
	return out, nil
}

func LinkedDiscordRoleIDs(ctx context.Context) ([]string, error) {
	linked, err := linkedRoles(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(linked))
	for discordRoleID := range linked {
		out = append(out, discordRoleID)
	}
	return out, nil
}
